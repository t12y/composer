package composer

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Service struct {
	id           int
	isDependency bool
	name         string
	readyOn      string
	workdir      string
	command      string
	dependsOn    []string
	environment  map[string]string
	killTimeout  time.Duration

	logPrefix string

	error     chan error
	ready     chan bool
	readyOnce sync.Once
	cmd       *exec.Cmd
}

func NewService(id int, name string, globalEnv Environment, cfg ServiceConfig) (*Service, error) {
	service := &Service{
		id:          id,
		name:        name,
		command:     cfg.Command,
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

func (s *Service) initCmd() error {
	if len(s.command) == 0 {
		return fmt.Errorf("command required")
	}

	s.cmd = exec.Command("/bin/sh", "-c", s.command)

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
