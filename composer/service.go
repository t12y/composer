package composer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/shlex"
)

type Service struct {
	id          int
	name        string
	readyOn     string
	workdir     string
	command     []string
	dependsOn   []string
	environment map[string]string
	killTimeout time.Duration

	logPrefix string

	error     chan error
	ready     chan bool
	readyOnce sync.Once
	cmd       *exec.Cmd
}

func NewService(id int, name string, globalEnv Environment, cfg ServiceConfig) (*Service, error) {
	command, err := shlex.Split(cfg.Command)
	if err != nil {
		return nil, fmt.Errorf("cannot parse command: %w", err)
	}

	service := &Service{
		id:          id,
		name:        name,
		command:     command,
		workdir:     cfg.Workdir,
		readyOn:     cfg.ReadyOn,
		dependsOn:   cfg.DependsOn,
		environment: cfg.Environment.Extends(globalEnv),
		killTimeout: time.Duration(cfg.KillTimeout) * time.Second,
		ready:       make(chan bool, 1),
		error:       make(chan error, 1),
	}

	if service.killTimeout == 0 {
		service.killTimeout = DefaultKillTimeout
	}

	if service.readyOn == "" {
		service.ready <- true
	}

	return service, nil
}

// cleanCommand returns a command without empty components
func cleanCommand(cmd []string) []string {
	cleanCmd := make([]string, 0, len(cmd))

	for i := range cmd {
		part := strings.TrimSpace(cmd[i])

		if part == "" {
			continue
		}

		cleanCmd = append(cleanCmd, part)
	}

	return cleanCmd
}

func (s *Service) initCmd() error {
	if len(s.command) == 0 {
		return fmt.Errorf("command required")
	}

	// expand environment variables in all components of the command
	for i := range s.command {
		s.command[i] = os.Expand(s.command[i], func(key string) string {
			return os.ExpandEnv(s.environment[key])
		})
	}

	s.command = cleanCommand(s.command)

	// support ~ substitute for HOME directory
	programName := s.command[0]
	if strings.HasPrefix(programName, "~") {
		programName = s.environment["HOME"] + programName[1:]
	}

	// determine absolute path to the command
	program, err := exec.LookPath(programName)
	if err != nil {
		return fmt.Errorf("program not found %s", programName)
	}

	s.cmd = exec.Command(program, s.command[1:]...)

	// set pgid, so we can terminate all subprocesses as well
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// set workdir
	s.cmd.Dir = s.workdir

	// set environment variables
	for key, value := range s.environment {
		value = os.ExpandEnv(value)
		env := fmt.Sprintf("%s=%s", key, value)
		s.cmd.Env = append(s.cmd.Env, env)
	}

	return nil
}
