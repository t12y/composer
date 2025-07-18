package composer_test

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/t12y/composer/composer"
)

func captureStdoutStderr(fn func()) string {
	originalStdout, originalStderr := os.Stdout, os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout, os.Stderr = w, w

	result := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		result <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout, os.Stderr = originalStdout, originalStderr

	return <-result
}

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
			"b1": {Command: "sleep 0.3 && echo 'ok'", ReadyOn: "ok"},
			"s1": {Command: "sleep 0.2"},
			"s2": {Command: "sleep 0.1", DependsOn: []string{"b1"}},
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

	const doneAfter = 300 * time.Millisecond
	if time.Since(start) < doneAfter {
		t.Errorf("process should be done after %v milliseconds, it took %v instead", doneAfter, time.Since(start))
	}
}

func TestLargeOutput(t *testing.T) {
	// outputLength should be > 64k to be sure we can support arbitrarily large outputs
	// see: `MaxScanTokenSize` value at https://pkg.go.dev/bufio#pkg-constants
	const outputLength = 70_000

	expectedOutput := strings.Repeat("0", outputLength)
	cmd := "printf '%0" + strconv.Itoa(outputLength) + "d' 0"

	cfg := composer.Config{
		Version:  composer.Version,
		Services: map[string]composer.ServiceConfig{"s1": {Command: cmd}},
	}

	c, err := composer.New(cfg, "s1")
	if err != nil {
		t.Errorf("error: %v", err)
	}

	// c.EnableDebug()

	output := captureStdoutStderr(func() { err = c.Run() })
	if err != nil {
		if !strings.Contains(err.Error(), "interrupted by user") {
			t.Errorf("error running composer: %v", err)
		}
	}

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("expected large output value not found in actual execution output:\nwant: '%s'\ngot '%s'", expectedOutput, output)
	}
}

func TestDaemonDependencies(t *testing.T) {
	const expectedOutput = "hello world"
	const unexpectedOutput = "goodbye cruel world"

	cfg := composer.Config{
		Version: composer.Version,
		Services: map[string]composer.ServiceConfig{
			"s1": {
				// the main service will exit immediately - however, it should wait on all dependencies to become ready
				// before executing, and then any remaining dependency services can be killed
				Command:   "echo 'hello from s1'",
				DependsOn: []string{"d1"},
			},
			"d1": {
				// the first child dependency will exit after 2 seconds but is ready immediately - this simulates a
				// dependency that runs forever in the background and should be killed after the main service exits
				Command:   "sleep 2 && echo '" + unexpectedOutput + "'",
				DependsOn: []string{"d2"},
			},
			"d2": {
				// the first grandchild dependency will signal ready after 1 second and then exit immediately - this
				// ensures the main service actually blocks on all dependencies to become ready before executing
				ReadyOn: expectedOutput,
				Command: "sleep 1 && echo '" + expectedOutput + "'",
			},
		},
	}

	c, err := composer.New(cfg, "s1")
	if err != nil {
		t.Errorf("error: %v", err)
	}

	// c.EnableDebug()

	output := captureStdoutStderr(func() { err = c.Run() })
	if err != nil {
		if !strings.Contains(err.Error(), "interrupted by user") {
			t.Errorf("error running composer: %v", err)
		}
	}

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("expected output value not found in actual execution output:\nwant: '%s'\ngot '%s'", expectedOutput, output)
	}

	if strings.Contains(output, unexpectedOutput) {
		t.Errorf("unexpected output value found in actual execution output:\nunexpected: '%s'\ngot '%s'", unexpectedOutput, output)
	}
}
