package composer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Version defines the highest supported version of the config
const Version = 1

// ParseConfig parses composer config file
// For available options, see definition of Config (`yaml` tags and comments)
func ParseConfig(filePath string) (*Config, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening composer file: %w", err)
	}

	cfg := new(Config)

	if err = yaml.NewDecoder(fp).Decode(cfg); err != nil {
		return nil, fmt.Errorf("error decoding composer file: %w", err)
	}

	if cfg.Version > Version {
		return nil, fmt.Errorf("composer needs to be updated")
	}

	var absFilePath string
	absFilePath, err = filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot determine absolute path to the config file: %w", err)
	}

	cfg.initEnvironment(filepath.Dir(absFilePath))

	return cfg, nil
}

// Config defines root config structure
type Config struct {
	// Version defines supported config version number
	Version int `yaml:"version"`

	// Environment defines global environmental variables available to all services.
	// It's possible to use $KEY notation, to use KEY value from current environment.
	Environment Environment `yaml:"environment"`

	// Services defines a map of service name to its configuration
	Services map[string]ServiceConfig `yaml:"services"`
}

// initEnvironment initializes global environment variable with default values
func (cfg *Config) initEnvironment(pwd string) {
	if cfg.Environment == nil {
		cfg.Environment = make(Environment)
	}

	if cfg.Environment["PWD"] == "" {
		cfg.Environment["PWD"] = pwd
	}

	for _, key := range []string{"PATH", "HOME"} {
		if cfg.Environment[key] == "" {
			cfg.Environment[key] = os.Getenv(key)
		}
	}
}

// ServicesToStart returns a list of services and dependencies to start
// (in order in which they should be started).
func (cfg *Config) ServicesToStart(initServices ...string) ([]string, error) {
	toStart, err := cfg.identifyServicesToBeStarted(initServices...)
	if err != nil {
		return nil, err
	}

	// sort services in order of execution and detect any circular dependencies
	result := make([]string, 0, len(toStart))
	resolved := make(map[string]bool)

	for len(resolved) < len(toStart) {
		anyServiceResolved := false

		for _, serviceName := range toStart {
			serviceDependenciesResolved := true

			if resolved[serviceName] {
				continue
			}

			service := cfg.Services[serviceName]

			for _, dependency := range service.DependsOn {
				if resolved[dependency] {
					continue
				} else {
					serviceDependenciesResolved = false
					break
				}
			}

			if serviceDependenciesResolved {
				resolved[serviceName] = true
				result = append(result, serviceName)
				anyServiceResolved = true
			}
		}

		if !anyServiceResolved {
			return nil, fmt.Errorf("circular service dependency detected")
		}
	}

	return result, nil
}

func (cfg *Config) identifyServicesToBeStarted(services ...string) ([]string, error) {
	toProcess := services
	processed := make(map[string]bool)
	toStart := make([]string, 0, len(cfg.Services))

	for len(toProcess) > 0 {
		var serviceName string

		// take out the first unprocessed service
		serviceName, toProcess = toProcess[0], toProcess[1:]

		if processed[serviceName] {
			continue
		}

		// check if the service exists and get its config
		serviceCfg, ok := cfg.Services[serviceName]
		if !ok {
			return nil, fmt.Errorf("unknown service: %s", serviceName)
		}

		for _, dependency := range serviceCfg.DependsOn {
			// don't process the same service twice
			if processed[dependency] {
				continue
			}

			// append to processing queue
			toProcess = append(toProcess, dependency)
		}

		// insert service before others
		toStart = append([]string{serviceName}, toStart...)

		// mark as processed
		processed[serviceName] = true
	}

	return toStart, nil
}

const DefaultKillTimeout = 5 * time.Second

// ServiceConfig defines a services configuration
type ServiceConfig struct {
	// Command defines which program to execute to start the service (REQUIRED).
	Command string `yaml:"command"`

	// Workdir defines working directory where the Command will be executed.
	// When empty, Command will run in the current working directory.
	Workdir string `yaml:"workdir"`

	// ReadyOn defines a match string for stdout/stderr which determines whether the service is ready or not.
	// When empty, services will be deemed ready immediately after start.
	ReadyOn string `yaml:"ready_on"`

	// DependsOn defines which other services should be started before this one.
	// When empty, service can start immediately.
	DependsOn []string `yaml:"depends_on"`

	// Environment defines environmental variables to be set when running the Command.
	// PWD, HOME and PATH are always set.
	// It's possible to use $KEY notation, to use KEY value from current environment.
	Environment Environment `yaml:"environment"`

	// KillTimeout defines maximum allowed duration for the process to shut down gracefully (before KILL signal is sent)
	// If not set, default of 5 seconds will be used.
	KillTimeout int `yaml:"kill_timeout"`
}

// Environment defines map of environmental keys to variables
type Environment map[string]string

// Extends returns Environment with keys and values from both `env` and `parent` environments.
// If `env` contains the same key as `parent`, value from `env` will be used.
func (env Environment) Extends(parent Environment) Environment {
	result := make(Environment)

	for key, value := range parent {
		result[key] = value
	}

	for key, value := range env {
		result[key] = value
	}

	return result
}
