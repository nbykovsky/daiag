package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	claudeexec "daiag/internal/executor/claude"
	codexexec "daiag/internal/executor/codex"
	"daiag/internal/logging"
	"daiag/internal/runtime"
	"daiag/internal/starlarkdsl"
)

type workflowRunner struct {
	stdout io.Writer
}

func NewDefault(stdout, stderr io.Writer) *App {
	return New(stdout, stderr, workflowRunner{stdout: stdout})
}

func (r workflowRunner) Run(ctx context.Context, cfg RunConfig) error {
	workdir, err := resolveWorkdir(cfg.Workdir)
	if err != nil {
		return err
	}

	workflowPath, err := filepath.Abs(cfg.Workflow)
	if err != nil {
		return fmt.Errorf("resolve workflow path: %w", err)
	}
	workflowBaseDir := filepath.Dir(workflowPath)

	inputs := runConfigInputs(cfg)
	loader := starlarkdsl.Loader{
		Params:  cfg.Params,
		Inputs:  inputs,
		BaseDir: workflowBaseDir,
	}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		return err
	}

	logger := logging.New(r.stdout)
	engine := runtime.Engine{
		Executors: map[string]runtime.Executor{
			"codex":  codexexec.New(),
			"claude": claudeexec.New(),
		},
		Logger: logger,
	}

	return engine.Run(ctx, runtime.RunInput{
		Workflow:     wf,
		WorkflowPath: cfg.Workflow,
		BaseDir:      workflowBaseDir,
		Workdir:      workdir,
		Inputs:       anyInputs(inputs),
	})
}

func resolveWorkdir(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("--workdir is required")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("--workdir must be an absolute path")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create workdir %q: %w", path, err)
	}
	return path, nil
}

func runConfigInputs(cfg RunConfig) map[string]string {
	if cfg.Inputs != nil {
		return cfg.Inputs
	}
	return cfg.Params
}

func anyInputs(inputs map[string]string) map[string]any {
	values := make(map[string]any, len(inputs))
	for key, value := range inputs {
		values[key] = value
	}
	return values
}
