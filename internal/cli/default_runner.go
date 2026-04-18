package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	claudeexec "daiag/internal/executor/claude"
	codexexec "daiag/internal/executor/codex"
	"daiag/internal/initcmd"
	"daiag/internal/logging"
	"daiag/internal/runtime"
	"daiag/internal/starlarkdsl"
)

type workflowRunner struct {
	stdout io.Writer
}

func NewDefault(stdout, stderr io.Writer) *App {
	return New(stdout, stderr, workflowRunner{stdout: stdout}, workflowValidator{})
}

type workflowValidator struct{}

func (v workflowValidator) Validate(_ context.Context, cfg ValidateConfig) error {
	projectDir, err := resolveProjectDir(cfg.ProjectDir)
	if err != nil {
		return err
	}
	workflowsLib, err := resolveWorkflowsLib(projectDir, cfg.WorkflowsLib)
	if err != nil {
		return err
	}
	workflowPath, err := starlarkdsl.ResolveWorkflowID(workflowsLib, cfg.Workflow)
	if err != nil {
		return err
	}
	loader := starlarkdsl.Loader{BaseDir: workflowsLib}
	loader.Inputs = cfg.Inputs
	loader.Params = cfg.Inputs
	_, err = loader.Load(workflowPath)
	return err
}

func (r workflowRunner) Run(ctx context.Context, cfg RunConfig) error {
	projectDir, err := resolveProjectDir(cfg.ProjectDir)
	if err != nil {
		return err
	}

	workflowsLib, err := resolveWorkflowsLib(projectDir, cfg.WorkflowsLib)
	if err != nil {
		return err
	}
	runDir, err := resolveRunDir(projectDir, cfg.RunDir, cfg.Workflow)
	if err != nil {
		return err
	}
	result, err := r.executeWorkflow(ctx, cfg.Workflow, projectDir, runDir, workflowsLib, cfg.Inputs)
	if err != nil {
		return err
	}
	return printWorkflowOutputs(r.stdout, result)
}

func (r workflowRunner) Bootstrap(ctx context.Context, cfg BootstrapConfig) error {
	projectDir, err := resolveProjectDir(cfg.ProjectDir)
	if err != nil {
		return err
	}
	workflowsLib, err := resolveWorkflowsLib(projectDir, cfg.WorkflowsLib)
	if err != nil {
		return err
	}
	if !pathWithin(projectDir, workflowsLib) {
		return fmt.Errorf("--workflows-lib %q must be inside --projectdir %q for bootstrap", workflowsLib, projectDir)
	}
	runDir, err := resolveRunDir(projectDir, cfg.RunDir, "bootstrap")
	if err != nil {
		return err
	}
	description, err := readBootstrapDescription(cfg)
	if err != nil {
		return err
	}

	result, err := r.executeWorkflow(ctx, cfg.Workflow, projectDir, runDir, workflowsLib, map[string]string{
		"description":   description,
		"workflows_lib": workflowsLib,
	})
	if err != nil {
		return err
	}
	workflowID, workflowPath, err := validateBootstrapResult(result, workflowsLib)
	if err != nil {
		return err
	}
	if err := validateGeneratedWorkflow(workflowsLib, workflowID); err != nil {
		return err
	}

	fmt.Fprintln(r.stdout, "bootstrap complete")
	fmt.Fprintf(r.stdout, "workflow: %s\n", workflowID)
	fmt.Fprintf(r.stdout, "workflow path: %s\n", workflowPath)
	fmt.Fprintf(r.stdout, "run dir: %s\n", result.RunDir)
	return nil
}

func (r workflowRunner) Init(_ context.Context, cfg InitConfig) error {
	target := cfg.ProjectDir
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
		target = cwd
	} else {
		abs, err := filepath.Abs(cfg.ProjectDir)
		if err != nil {
			return fmt.Errorf("resolve --projectdir: %w", err)
		}
		target = abs
	}
	return initcmd.Init(initcmd.Config{
		ProjectDir: target,
		Workflows:  cfg.Workflows,
		Force:      cfg.Force,
	}, r.stdout)
}

func (r workflowRunner) ListWorkflows() []string {
	return initcmd.AvailableWorkflows()
}

func (r workflowRunner) executeWorkflow(ctx context.Context, workflowID string, projectDir string, runDir string, workflowsLib string, inputs map[string]string) (*runtime.RunResult, error) {
	workflowPath, err := starlarkdsl.ResolveWorkflowID(workflowsLib, workflowID)
	if err != nil {
		return nil, err
	}

	loader := starlarkdsl.Loader{
		Params:  inputs,
		Inputs:  inputs,
		BaseDir: workflowsLib,
	}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		return nil, err
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
		ProjectDir:   projectDir,
		RunDir:       runDir,
		Inputs:       anyInputs(inputs),
	})
}

func resolveProjectDir(path string) (string, error) {
	if path != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve --projectdir: %w", err)
		}
		return cleanExistingDirFlag("--projectdir", absPath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	projectDir, err := findProjectRoot(cwd)
	if err != nil {
		return "", err
	}
	return cleanExistingDirFlag("--projectdir", projectDir)
}

func resolveWorkflowsLib(projectDir string, path string) (string, error) {
	workflowsLib := path
	if workflowsLib == "" {
		workflowsLib = filepath.Join(projectDir, ".daiag", "workflows")
	} else if !filepath.IsAbs(workflowsLib) {
		workflowsLib = filepath.Join(projectDir, workflowsLib)
	}
	absPath, err := filepath.Abs(workflowsLib)
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
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve --workflows-lib symlinks: %w", err)
	}
	return filepath.Clean(resolved), nil
}

func resolveRunDir(projectDir string, path string, workflowID string) (string, error) {
	if path != "" {
		runDir := path
		if !filepath.IsAbs(runDir) {
			runDir = filepath.Join(projectDir, runDir)
		}
		absPath, err := filepath.Abs(runDir)
		if err != nil {
			return "", fmt.Errorf("resolve --run-dir: %w", err)
		}
		if err := os.MkdirAll(absPath, 0o755); err != nil {
			return "", fmt.Errorf("create --run-dir %q: %w", absPath, err)
		}
		resolved, err := cleanExistingDirFlag("--run-dir", absPath)
		if err != nil {
			return "", err
		}
		if !pathWithin(projectDir, resolved) {
			return "", fmt.Errorf("--run-dir %q must be inside --projectdir %q", resolved, projectDir)
		}
		return resolved, nil
	}

	return createDefaultRunDir(projectDir, workflowID)
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

func createDefaultRunDir(projectDir string, workflowID string) (string, error) {
	if workflowID == "" {
		return "", fmt.Errorf("workflow ID is required for default run dir")
	}
	for attempt := 0; attempt < 10; attempt++ {
		base := filepath.Join(projectDir, ".daiag", "runs", workflowID, runTimestamp(time.Now().UTC()))
		path, err := createUniqueRunDir(base)
		if err == nil {
			return path, nil
		}
		if !os.IsExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("create default run dir: exhausted timestamp retries")
}

func createUniqueRunDir(base string) (string, error) {
	for suffix := 0; suffix < 100; suffix++ {
		path := base
		if suffix > 0 {
			path = fmt.Sprintf("%s-%02d", base, suffix)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", fmt.Errorf("create run dir parent %q: %w", filepath.Dir(path), err)
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", fmt.Errorf("create run dir %q: %w", path, err)
		}
		return cleanExistingDirFlag("--run-dir", path)
	}
	return "", fmt.Errorf("%w: run dir %q already exists with suffixes 00-99", os.ErrExist, base)
}

func runTimestamp(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("%s-%09dZ", t.Format("20060102-150405"), t.Nanosecond())
}

func readBootstrapDescription(cfg BootstrapConfig) (string, error) {
	if cfg.Description != "" {
		return cfg.Description, nil
	}
	path := cfg.DescriptionFile
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve --description-file: %w", err)
		}
		path = absPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read --description-file %q: %w", path, err)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("--description-file %q is not valid UTF-8", path)
	}
	return string(data), nil
}

func validateBootstrapResult(result *runtime.RunResult, workflowsLib string) (string, string, error) {
	if result == nil {
		return "", "", fmt.Errorf("bootstrap workflow returned no result")
	}
	workflowID, err := requiredStringResult(result, "workflow_id")
	if err != nil {
		return "", "", err
	}
	workflowPath, err := requiredStringResult(result, "workflow_path")
	if err != nil {
		return "", "", err
	}
	outcome, err := requiredStringResult(result, "outcome")
	if err != nil {
		return "", "", err
	}
	if outcome != "complete" {
		return "", "", fmt.Errorf("bootstrap outcome = %q, want complete", outcome)
	}
	if !filepath.IsAbs(workflowPath) {
		return "", "", fmt.Errorf("bootstrap result workflow_path %q must be absolute", workflowPath)
	}

	expectedPath := filepath.Join(workflowsLib, workflowID, "workflow.star")
	resolvedExpected, err := filepath.EvalSymlinks(expectedPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve expected generated workflow path %q: %w", expectedPath, err)
	}
	resolvedActual, err := filepath.EvalSymlinks(workflowPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve bootstrap workflow_path %q: %w", workflowPath, err)
	}
	resolvedExpected = filepath.Clean(resolvedExpected)
	resolvedActual = filepath.Clean(resolvedActual)
	if resolvedActual != resolvedExpected {
		return "", "", fmt.Errorf("bootstrap result workflow_path %q must match %q", resolvedActual, resolvedExpected)
	}

	return workflowID, resolvedActual, nil
}

func requiredStringResult(result *runtime.RunResult, key string) (string, error) {
	value, ok := result.OutputResults[key]
	if !ok {
		return "", fmt.Errorf("bootstrap output result %q is required", key)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("bootstrap output result %q must be a string", key)
	}
	if text == "" {
		return "", fmt.Errorf("bootstrap output result %q must not be empty", key)
	}
	return text, nil
}

func validateGeneratedWorkflow(workflowsLib string, workflowID string) error {
	workflowPath, err := starlarkdsl.ResolveWorkflowID(workflowsLib, workflowID)
	if err != nil {
		return fmt.Errorf("validate generated workflow %q: %w", workflowID, err)
	}
	loader := starlarkdsl.Loader{BaseDir: workflowsLib}
	if _, err := loader.Load(workflowPath); err != nil {
		return fmt.Errorf("validate generated workflow %q: %w", workflowID, err)
	}
	return nil
}

func cleanExistingDirFlag(flagName string, path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", flagName, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("%s %q: %w", flagName, absPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s %q is not a directory", flagName, absPath)
	}
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve %s symlinks: %w", flagName, err)
	}
	return filepath.Clean(resolved), nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func printWorkflowOutputs(w io.Writer, result *runtime.RunResult) error {
	if result == nil || (len(result.OutputArtifacts) == 0 && len(result.OutputResults) == 0) {
		return nil
	}
	if _, err := fmt.Fprintln(w, "workflow outputs:"); err != nil {
		return err
	}
	artifactKeys := make([]string, 0, len(result.OutputArtifacts))
	for key := range result.OutputArtifacts {
		artifactKeys = append(artifactKeys, key)
	}
	sort.Strings(artifactKeys)
	for _, key := range artifactKeys {
		if _, err := fmt.Fprintf(w, "artifact %s: %s\n", key, result.OutputArtifacts[key]); err != nil {
			return err
		}
	}
	resultKeys := make([]string, 0, len(result.OutputResults))
	for key := range result.OutputResults {
		resultKeys = append(resultKeys, key)
	}
	sort.Strings(resultKeys)
	for _, key := range resultKeys {
		data, err := json.Marshal(result.OutputResults[key])
		if err != nil {
			return fmt.Errorf("encode output result %q: %w", key, err)
		}
		if _, err := fmt.Fprintf(w, "result %s: %s\n", key, data); err != nil {
			return err
		}
	}
	return nil
}

func anyInputs(inputs map[string]string) map[string]any {
	values := make(map[string]any, len(inputs))
	for key, value := range inputs {
		values[key] = value
	}
	return values
}
