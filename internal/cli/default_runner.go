package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	workflowsLib, err := resolveWorkflowsLib(cfg.WorkflowsLib)
	if err != nil {
		return err
	}
	workflowPath, err := starlarkdsl.ResolveWorkflowID(workflowsLib, cfg.Workflow)
	if err != nil {
		return err
	}

	inputs := runConfigInputs(cfg)
	loader := starlarkdsl.Loader{
		Params:  cfg.Params,
		Inputs:  inputs,
		BaseDir: workflowsLib,
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
		WorkflowPath: workflowPath,
		BaseDir:      workflowsLib,
		Workdir:      workdir,
		Inputs:       anyInputs(inputs),
	})
}

func resolveWorkflowsLib(path string) (string, error) {
	if path != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve --workflows-lib: %w", err)
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return "", fmt.Errorf("--workflows-lib %q: %w", absPath, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("--workflows-lib %q is not a directory", absPath)
		}
		return absPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	projectDir, err := findProjectRoot(cwd)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectDir, ".daiag", "workflows"), nil
}

func findProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	walked := []string{}
	for {
		walked = append(walked, dir)
		daiagDir := filepath.Join(dir, ".daiag")
		info, err := os.Stat(daiagDir)
		if err == nil {
			if info.IsDir() {
				return dir, nil
			}
			return "", fmt.Errorf("default workflows library requires %q to be a directory", daiagDir)
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %q: %w", daiagDir, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("--workflows-lib omitted and no .daiag directory found from %q; walked: %s", startDir, strings.Join(walked, ", "))
		}
		dir = parent
	}
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
