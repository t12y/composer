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
