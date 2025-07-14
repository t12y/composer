package composer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Composer struct {
	cfg          Config
	waitFor      map[string]bool
	waitLock     sync.Mutex
	services     []*Service
	running      map[string]bool
	cleanupWait  sync.WaitGroup
	outputWait   sync.WaitGroup
	lastError    chan error
	debugEnabled bool
}

// New runs a service with all of its dependencies
func New(cfg Config, initServices ...string) (*Composer, error) {
	servicesToStart, err := cfg.ServicesToStart(initServices...)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	services := make([]*Service, len(servicesToStart))

	env := cfg.Environment

	for i, name := range servicesToStart {
		if services[i], err = NewService(i, name, env, cfg.Services[name]); err != nil {
			return nil, fmt.Errorf("error setting up service %s: %w", name, err)
		}
	}

	composer := &Composer{
		cfg:          cfg,
		services:     services,
		debugEnabled: os.Getenv("DEBUG") != "",
		lastError:    make(chan error, len(servicesToStart)),
	}

	return composer, nil
}

// Run starts all services
func (c *Composer) Run() error {
	c.info("Preparing composer")

	const maxOpenFiles = 65000
	opeFilesRLimit := &syscall.Rlimit{
		Cur: maxOpenFiles,
		Max: maxOpenFiles,
	}

	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, opeFilesRLimit); err != nil {
		return fmt.Errorf("error setting system limits: %w", err)
	}

	if err := c.prepareServices(); err != nil {
		return fmt.Errorf("error preparing services: %w", err)
	}

	defer c.cleanup()

	c.info("Starting services")
	if err := c.startServices(); err != nil {
		return err
	}

	return nil
}

// RunAll runs all provided services and only exists when one of services exit with an error or all successful.
func (c *Composer) RunAll(services ...string) error {
	c.waitFor = make(map[string]bool)

	for _, service := range services {
		c.waitFor[service] = true
	}

	return c.Run()
}

// Interrupt interrupts composer execution
func (c *Composer) Interrupt() {
	c.info("Interrupting composer...")
	c.lastError <- fmt.Errorf("interrupted by user")
}

// EnableDebug enables debug logging
func (c *Composer) EnableDebug() {
	c.debugEnabled = true
}

func (c *Composer) prepareServices() error {
	for i := range c.services {
		service := c.services[i]

		c.debug("Preparing service: %s", service.name)

		if err := c.prepareService(service); err != nil {
			return fmt.Errorf("cannot initialize service %s: %w", service.name, err)
		}
	}

	return nil
}

func (c *Composer) prepareService(service *Service) error {
	if err := service.initCmd(); err != nil {
		return err
	}

	c.debug("cmd: %s", strings.Join(service.cmd.Args, " "))
	c.debug("dir: %s", service.cmd.Dir)
	c.debug("env: %v", service.cmd.Env)

	c.debug("Registering outputs")
	if err := c.registerOutputs(service); err != nil {
		return fmt.Errorf("error registering stdout: %w", err)
	}

	return nil
}

const terminalResetColor = "\033[0m"

var terminalColors = []string{
	"\033[31m", // red
	"\033[32m", // green
	"\033[33m", // yellow
	"\033[34m", // blue
	"\033[35m", // purple
	"\033[36m", // cyan
}

func (c *Composer) registerOutputs(service *Service) error {
	color := terminalColors[service.id%len(terminalColors)]
	longestServiceName := len(service.name)

	for i := range c.services {
		if longestServiceName < len(c.services[i].name) {
			longestServiceName = len(c.services[i].name)
		}
	}

	prefixSpace := strings.Repeat(" ", longestServiceName-len(service.name))

	service.logPrefix = fmt.Sprintf("\r%s["+service.name+"]%s |%s", color, prefixSpace, terminalResetColor)

	if err := c.registerOutput(service, service.cmd.StdoutPipe, os.Stdout); err != nil {
		return fmt.Errorf("cannot register stdout: %w", err)
	}

	if err := c.registerOutput(service, service.cmd.StderrPipe, os.Stderr); err != nil {
		return fmt.Errorf("cannot register stderr: %w", err)
	}

	return nil
}

func (c *Composer) registerOutput(service *Service, readerFn func() (io.ReadCloser, error), writer io.Writer) error {
	reader, err := readerFn()
	if err != nil {
		return fmt.Errorf("cannot get reader: %w", err)
	}

	c.outputWait.Add(1)

	bufReader := bufio.NewReader(reader)

	go func() {
		defer c.outputWait.Done()

		lastLine := ""

		for line := ""; err == nil; {
			line, err = bufReader.ReadString('\n')
			line = strings.TrimRight(line, "\r\n")

			// skip repeated lines
			if strings.EqualFold(line, lastLine) {
				continue
			}

			lastLine = line

			_, _ = fmt.Fprintln(writer, service.logPrefix, line)

			if service.readyOn != "" && strings.Contains(line, service.readyOn) {
				service.readyOnce.Do(func() {
					service.ready <- true
				})
			}
		}

		if err != io.EOF {
			_, _ = fmt.Fprintln(os.Stderr, service.name, fmt.Sprintf("reader error: %v", err))
		}
	}()

	return nil
}

func (c *Composer) startServices() error {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGHUP)

	for i := range c.services {
		service := c.services[i]

		c.info("Starting service %s", service.name)
		if err := service.cmd.Start(); err != nil {
			return fmt.Errorf("error starting service %s: %w", service.name, err)
		}

		go func() {
			c.debug("waiting for: %s", service.name)
			// we must first wait for command stdout / stderr because cmd.Wait() will close pipes after seeing the command exit
			// see: https://pkg.go.dev/os/exec#Cmd.StdoutPipe
			c.outputWait.Wait()
			err := service.cmd.Wait()
			c.debug("wait-err from %s: %v", service.name, err)
			c.quit(service.name, err)
		}()

		c.info("Waiting for service %s to be ready", service.name)
		select {
		case <-service.ready:
			c.debug("%s is ready", service.name)
		case <-signalCh:
			c.Interrupt()
		case err := <-service.error:
			c.debug("service %s error: %v", service.name, err)
			return err
		case err := <-c.lastError:
			c.debug("global (service) error: %v", err)
			return err
		}
	}

	c.debug("all services running")

	for {
		select {
		case <-signalCh:
			c.Interrupt()
		case err := <-c.lastError:
			c.debug("global error: %v", err)
			return err
		}
	}
}

func (c *Composer) info(msg string, args ...interface{}) {
	fmt.Println("[composer]", fmt.Sprintf(msg, args...))
}

func (c *Composer) debug(msg string, args ...interface{}) {
	if !c.debugEnabled {
		return
	}

	fmt.Println("[composer-debug]", fmt.Sprintf(msg, args...))
}

func (c *Composer) quit(serviceName string, err error) {
	c.waitLock.Lock()
	defer c.waitLock.Unlock()

	c.debug("waitFor length: %d", len(c.waitFor))
	if len(c.waitFor) > 0 && err == nil && c.waitFor[serviceName] {
		c.debug("service %s exited cleanly", serviceName)
		delete(c.waitFor, serviceName)

		if len(c.waitFor) > 0 {
			c.debug("%d more services to wait for (%v)", len(c.waitFor), c.waitFor)
			return
		}
	}

	c.debug("service %s quit with error: %v", serviceName, err)
	c.lastError <- err
}

func (c *Composer) cleanup() {
	c.debug("cleanup")

	c.cleanupWait.Add(len(c.services))

	for i := range c.services {
		go c.cleanupService(c.services[i])
	}

	c.cleanupWait.Wait()
}

func (c *Composer) cleanupService(service *Service) {
	defer c.cleanupWait.Done()

	c.debug("cleanup %s", service.name)

	if service.cmd == nil {
		c.debug("cleanup %s - cmd nil", service.name)
		return
	}

	if service.cmd.ProcessState != nil && service.cmd.ProcessState.Exited() {
		c.debug("cleanup %s - already exited", service.name)
		return
	}

	if service.cmd.Process == nil {
		c.debug("cleanup %s - no process info", service.name)
		return
	}

	pid := service.cmd.Process.Pid

	killTimer := time.AfterFunc(service.killTimeout, func() {
		c.debug("cleanup %s - killing %d", service.name, pid)
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error killing service %s with PID %d\n", service.name, pid)
		}
	})
	defer killTimer.Stop()

	c.debug("cleanup %s - interrupting %d", service.name, pid)
	if err := syscall.Kill(-pid, syscall.SIGINT); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error interrupting service %s with PID %d\n", service.name, pid)
	}

	if _, err := service.cmd.Process.Wait(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error waiting for service %s with PID %d to die\n", service.name, pid)
	}
}
