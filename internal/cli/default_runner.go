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

	loader := starlarkdsl.Loader{
		Params:  cfg.Params,
		BaseDir: workdir,
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
		BaseDir:      workdir,
		Workdir:      workdir,
	})
}

func resolveWorkdir(path string) (string, error) {
	if path == "" {
		return os.Getwd()
	}
	return filepath.Abs(path)
}
