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
		"--projectdir", "/repo",
		"--run-dir", ".daiag/runs/poem/1",
		"--workflows-lib", "/shared/workflows",
		"--input", "feature=poem",
		"--input", "mode=fast",
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
	if runner.cfg.ProjectDir != "/repo" {
		t.Fatalf("projectdir = %q, want %q", runner.cfg.ProjectDir, "/repo")
	}
	if runner.cfg.RunDir != ".daiag/runs/poem/1" {
		t.Fatalf("run-dir = %q, want %q", runner.cfg.RunDir, ".daiag/runs/poem/1")
	}
	if !runner.cfg.Verbose {
		t.Fatal("verbose = false, want true")
	}
	if got := runner.cfg.Inputs["feature"]; got != "poem" {
		t.Fatalf("input feature = %q, want %q", got, "poem")
	}
	if got := runner.cfg.Inputs["mode"]; got != "fast" {
		t.Fatalf("input mode = %q, want %q", got, "fast")
	}
}

func TestAppRunRejectsParamFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
		"--param", "name=rain",
	})

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), `unknown flag: --param`) {
		t.Fatalf("stderr = %q, want unknown --param error", stderr.String())
	}
}

func TestAppRunPropagatesRunnerError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{err: errors.New("boom")}, nil)
	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "poem",
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
		"--projectdir", "/repo",
		"--workflows-lib", "/shared/workflows",
		"--input", "name=rain",
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
	if v.cfg.ProjectDir != "/repo" {
		t.Fatalf("projectdir = %q, want %q", v.cfg.ProjectDir, "/repo")
	}
	if got := v.cfg.Inputs["name"]; got != "rain" {
		t.Fatalf("input name = %q, want %q", got, "rain")
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

func TestAppBootstrapInvokesBootstrapper(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runner := &fakeRunner{}
	app := New(&stdout, &stderr, runner, nil)
	exitCode := app.Run(context.Background(), []string{
		"bootstrap",
		"--description", "create a workflow",
		"--workflow", "custom_bootstrap",
		"--projectdir", "/repo",
		"--run-dir", ".daiag/runs/bootstrap/1",
		"--workflows-lib", ".daiag/workflows",
		"--verbose",
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !runner.bootstrapCalled {
		t.Fatal("bootstrapper was not called")
	}
	if runner.bootstrapCfg.Workflow != "custom_bootstrap" {
		t.Fatalf("workflow = %q, want custom_bootstrap", runner.bootstrapCfg.Workflow)
	}
	if runner.bootstrapCfg.Description != "create a workflow" {
		t.Fatalf("description = %q, want create a workflow", runner.bootstrapCfg.Description)
	}
	if runner.bootstrapCfg.ProjectDir != "/repo" {
		t.Fatalf("projectdir = %q, want /repo", runner.bootstrapCfg.ProjectDir)
	}
	if runner.bootstrapCfg.RunDir != ".daiag/runs/bootstrap/1" {
		t.Fatalf("run-dir = %q, want .daiag/runs/bootstrap/1", runner.bootstrapCfg.RunDir)
	}
	if runner.bootstrapCfg.WorkflowsLib != ".daiag/workflows" {
		t.Fatalf("workflows-lib = %q, want .daiag/workflows", runner.bootstrapCfg.WorkflowsLib)
	}
	if !runner.bootstrapCfg.Verbose {
		t.Fatal("verbose = false, want true")
	}
}

func TestAppBootstrapDefaultsWorkflow(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runner := &fakeRunner{}
	app := New(&stdout, &stderr, runner, nil)
	exitCode := app.Run(context.Background(), []string{
		"bootstrap",
		"--description", "create a workflow",
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if runner.bootstrapCfg.Workflow != "workflow_bootstrapper" {
		t.Fatalf("workflow = %q, want workflow_bootstrapper", runner.bootstrapCfg.Workflow)
	}
}

func TestAppBootstrapRejectsDescriptionAndDescriptionFile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
	exitCode := app.Run(context.Background(), []string{
		"bootstrap",
		"--description", "create a workflow",
		"--description-file", "request.md",
	})

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "exactly one of --description or --description-file is required") {
		t.Fatalf("stderr = %q, want description conflict error", stderr.String())
	}
}

func TestAppInitList(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
	exitCode := app.Run(context.Background(), []string{"init", "--list"})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "workflow_bootstrapper") {
		t.Fatalf("stdout = %q, want workflow IDs", stdout.String())
	}
}

func TestAppInitInvokesInitializer(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runner := &fakeRunner{}
	app := New(&stdout, &stderr, runner, nil)
	exitCode := app.Run(context.Background(), []string{
		"init",
		"--workflow", "workflow_bootstrapper",
		"--workflow", "code_review_pipeline",
		"--force",
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !runner.initCalled {
		t.Fatal("Init was not called")
	}
	if !runner.initCfg.Force {
		t.Error("Force should be true")
	}
	if len(runner.initCfg.Workflows) != 2 {
		t.Errorf("Workflows = %v, want 2 entries", runner.initCfg.Workflows)
	}
}

func TestAppInitRejectsUnexpectedArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr, &fakeRunner{}, nil)
	exitCode := app.Run(context.Background(), []string{"init", "extra-arg"})

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
}

type fakeRunner struct {
	called          bool
	bootstrapCalled bool
	initCalled      bool
	cfg             RunConfig
	bootstrapCfg    BootstrapConfig
	initCfg         InitConfig
	err             error
}

func (f *fakeRunner) Run(_ context.Context, cfg RunConfig) error {
	f.called = true
	f.cfg = cfg
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeRunner) Bootstrap(_ context.Context, cfg BootstrapConfig) error {
	f.bootstrapCalled = true
	f.bootstrapCfg = cfg
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeRunner) Init(_ context.Context, cfg InitConfig) error {
	f.initCalled = true
	f.initCfg = cfg
	return f.err
}

func (f *fakeRunner) ListWorkflows() []string {
	return []string{"workflow_bootstrapper", "code_review_pipeline"}
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

type fakeInitializer struct {
	initCalled bool
	initCfg    InitConfig
	workflows  []string
	err        error
}

func (f *fakeInitializer) Init(_ context.Context, cfg InitConfig) error {
	f.initCalled = true
	f.initCfg = cfg
	return f.err
}

func (f *fakeInitializer) ListWorkflows() []string {
	return f.workflows
}
