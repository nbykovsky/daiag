package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAppRunMissingWorkflow(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{})
	exitCode := app.Run(context.Background(), []string{"run"})

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "--workflow is required") {
		t.Fatalf("stderr = %q, want workflow error", stderr.String())
	}
}

func TestAppRunInvokesRunner(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runner := &fakeRunner{}
	app := New(&stdout, &stderr, runner)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "workflows/poem.star",
		"--param", "name=rain",
		"--param", "mode=fast",
		"--workdir", "/tmp/work",
		"--verbose",
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !runner.called {
		t.Fatal("runner was not called")
	}
	if runner.cfg.Workflow != "workflows/poem.star" {
		t.Fatalf("workflow = %q, want %q", runner.cfg.Workflow, "workflows/poem.star")
	}
	if runner.cfg.Workdir != "/tmp/work" {
		t.Fatalf("workdir = %q, want %q", runner.cfg.Workdir, "/tmp/work")
	}
	if !runner.cfg.Verbose {
		t.Fatal("verbose = false, want true")
	}
	if got := runner.cfg.Params["name"]; got != "rain" {
		t.Fatalf("param name = %q, want %q", got, "rain")
	}
	if got := runner.cfg.Params["mode"]; got != "fast" {
		t.Fatalf("param mode = %q, want %q", got, "fast")
	}
}

func TestAppRunRejectsInvalidParam(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{})
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "workflows/poem.star",
		"--param", "invalid",
	})

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), `invalid --param "invalid"`) {
		t.Fatalf("stderr = %q, want param validation error", stderr.String())
	}
}

func TestAppRunPropagatesRunnerError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{err: errors.New("boom")})
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "workflows/poem.star",
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("stderr = %q, want runner error", stderr.String())
	}
}

type fakeRunner struct {
	called bool
	cfg    RunConfig
	err    error
}

func (f *fakeRunner) Run(_ context.Context, cfg RunConfig) error {
	f.called = true
	f.cfg = cfg
	if f.err != nil {
		return f.err
	}
	return nil
}
