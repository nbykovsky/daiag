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

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
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
	app := New(&stdout, &stderr, runner, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
		"--workflows-lib", "/shared/workflows",
		"--input", "feature=poem",
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
	if runner.cfg.Workflow != "poem" {
		t.Fatalf("workflow = %q, want %q", runner.cfg.Workflow, "poem")
	}
	if runner.cfg.WorkflowsLib != "/shared/workflows" {
		t.Fatalf("workflows lib = %q, want %q", runner.cfg.WorkflowsLib, "/shared/workflows")
	}
	if runner.cfg.Workdir != "/tmp/work" {
		t.Fatalf("workdir = %q, want %q", runner.cfg.Workdir, "/tmp/work")
	}
	if !runner.cfg.Verbose {
		t.Fatal("verbose = false, want true")
	}
	if got := runner.cfg.Inputs["feature"]; got != "poem" {
		t.Fatalf("input feature = %q, want %q", got, "poem")
	}
	if got := runner.cfg.Params["name"]; got != "rain" {
		t.Fatalf("param name = %q, want %q", got, "rain")
	}
	if got := runner.cfg.Params["mode"]; got != "fast" {
		t.Fatalf("param mode = %q, want %q", got, "fast")
	}
}

func TestAppRunMissingWorkdir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
	})

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "--workdir is required") {
		t.Fatalf("stderr = %q, want workdir error", stderr.String())
	}
}

func TestAppRunKeepsParamAsInputAlias(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runner := &fakeRunner{}
	app := New(&stdout, &stderr, runner, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
		"--workdir", "/tmp/work",
		"--param", "name=rain",
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if got := runner.cfg.Inputs["name"]; got != "rain" {
		t.Fatalf("input name = %q, want %q", got, "rain")
	}
	if got := runner.cfg.Params["name"]; got != "rain" {
		t.Fatalf("param name = %q, want %q", got, "rain")
	}
}

func TestAppRunRejectsConflictingInputAndParam(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
		"--input", "name=rain",
		"--param", "name=snow",
	})

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), `conflicting workflow input "name"`) {
		t.Fatalf("stderr = %q, want conflict error", stderr.String())
	}
}

func TestAppRunRejectsInvalidParam(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
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

	app := New(&stdout, &stderr, &fakeRunner{err: errors.New("boom")}, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
		"--workdir", "/tmp/work",
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("stderr = %q, want runner error", stderr.String())
	}
}

func TestAppValidateMissingWorkflow(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, nil, &fakeValidator{})
	exitCode := app.Run(context.Background(), []string{"validate"})
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "--workflow is required") {
		t.Fatalf("stderr = %q, want --workflow error", stderr.String())
	}
}

func TestAppValidateInvokesValidator(t *testing.T) {
	var stdout, stderr bytes.Buffer
	v := &fakeValidator{}
	app := New(&stdout, &stderr, nil, v)
	exitCode := app.Run(context.Background(), []string{
		"validate",
		"--workflow", "poem",
		"--workflows-lib", "/shared/workflows",
	})
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !v.called {
		t.Fatal("validator was not called")
	}
	if v.cfg.Workflow != "poem" {
		t.Fatalf("workflow = %q, want %q", v.cfg.Workflow, "poem")
	}
	if v.cfg.WorkflowsLib != "/shared/workflows" {
		t.Fatalf("workflows-lib = %q, want %q", v.cfg.WorkflowsLib, "/shared/workflows")
	}
}

func TestAppValidatePropagatesValidatorError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, nil, &fakeValidator{err: errors.New("boom")})
	exitCode := app.Run(context.Background(), []string{
		"validate",
		"--workflow", "poem",
	})
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("stderr = %q, want validator error", stderr.String())
	}
}

func TestAppValidateSuccessPrintsMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr, nil, &fakeValidator{})
	exitCode := app.Run(context.Background(), []string{
		"validate",
		"--workflow", "poem",
	})
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `workflow "poem" is valid`) {
		t.Fatalf("stdout = %q, want success message", stdout.String())
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

type fakeValidator struct {
	called bool
	cfg    ValidateConfig
	err    error
}

func (f *fakeValidator) Validate(_ context.Context, cfg ValidateConfig) error {
	f.called = true
	f.cfg = cfg
	return f.err
}
