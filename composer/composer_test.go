package composer_test

import (
	"strings"
	"testing"
	"time"

	"github.com/t12y/composer/composer"
)

func TestKill(t *testing.T) {
	cfg := composer.Config{
		Version: composer.Version,
		Services: map[string]composer.ServiceConfig{
			"test": {
				Command:     "sh -c \"(trap '' INT && sleep 5)\"",
				KillTimeout: 1,
			},
		},
	}

	c, err := composer.New(cfg, "test")
	if err != nil {
		t.Errorf("error: %v", err)
	}

	//c.EnableDebug()

	time.AfterFunc(time.Second, c.Interrupt)

	start := time.Now()

	if err = c.Run(); err != nil {
		if !strings.Contains(err.Error(), "interrupted by user") {
			t.Errorf("error running composer: %v", err)
		}
	}

	const killAfter = 3 * time.Second
	if time.Since(start) > killAfter {
		t.Errorf("process should be killed after %v seconds, it took %v instead", killAfter, time.Since(start))
	}
}

func TestRunAll(t *testing.T) {
	cfg := composer.Config{
		Version: composer.Version,
		Services: map[string]composer.ServiceConfig{
			"s1": {Command: "sleep 0.1"},
			"s2": {Command: "sleep 0.2"},
		},
	}

	c, err := composer.New(cfg, "s1", "s2")
	if err != nil {
		t.Errorf("error: %v", err)
	}

	// c.EnableDebug()

	start := time.Now()

	if err = c.RunAll("s1", "s2"); err != nil {
		if !strings.Contains(err.Error(), "interrupted by user") {
			t.Errorf("error running composer: %v", err)
		}
	}

	const doneAfter = 200 * time.Millisecond
	if time.Since(start) < doneAfter {
		t.Errorf("process should be done after %v seconds, it took %v instead", doneAfter, time.Since(start))
	}
}

func TestRunAll_DependencyExistsFirst(t *testing.T) {
	cfg := composer.Config{
		Version: composer.Version,
		Services: map[string]composer.ServiceConfig{
			"b1": {Command: "sleep 0.1"},
			"s1": {Command: "sleep 0.2"},
			"s2": {Command: "sleep 0.3", DependsOn: []string{"b1"}},
		},
	}

	c, err := composer.New(cfg, "s1", "s2")
	if err != nil {
		t.Errorf("error: %v", err)
	}

	c.EnableDebug()

	start := time.Now()

	if err = c.RunAll("s1", "s2"); err != nil {
		if !strings.Contains(err.Error(), "interrupted by user") {
			t.Errorf("error running composer: %v", err)
		}
	}

	const doneAfter = 150 * time.Millisecond
	if time.Since(start) > doneAfter {
		t.Errorf("process should be done after %v seconds, it took %v instead", doneAfter, time.Since(start))
	}
}
