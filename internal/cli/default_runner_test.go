package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultRunnerRunsInputSubworkflowWorkflow(t *testing.T) {
	workdir := t.TempDir()
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	writeCLITestWorkflow(t, workflowsLib)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent",
		"--workflows-lib", workflowsLib,
		"--input", "name=rain",
		"--workdir", workdir,
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	for _, fragment := range []string{
		"workflow start id=parent",
		"subworkflow start id=child workflow=child",
		"subworkflow done id=child artifacts=spec results=name",
		"workflow done id=parent status=success",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout missing %q:\n%s", fragment, stdout.String())
		}
	}
}

func TestDefaultRunnerLoadsWorkflowFromLibraryNotWorkdir(t *testing.T) {
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	workdir := t.TempDir()
	writeCLITestWorkflow(t, workflowsLib)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent",
		"--workflows-lib", workflowsLib,
		"--input", "name=rain",
		"--workdir", workdir,
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "subworkflow done id=child artifacts=spec results=name") {
		t.Fatalf("stdout missing child subworkflow success:\n%s", stdout.String())
	}
}

func TestDefaultRunnerKeepsParamAliasForInputWorkflow(t *testing.T) {
	workdir := t.TempDir()
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	writeCLITestWorkflow(t, workflowsLib)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent",
		"--workflows-lib", workflowsLib,
		"--param", "name=rain",
		"--workdir", workdir,
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "workflow done id=parent status=success") {
		t.Fatalf("stdout missing workflow success:\n%s", stdout.String())
	}
}

func TestDefaultRunnerReportsMissingWorkflowInput(t *testing.T) {
	workdir := t.TempDir()
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	writeCLITestWorkflow(t, workflowsLib)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent",
		"--workflows-lib", workflowsLib,
		"--workdir", workdir,
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), `missing workflow input "name"`) {
		t.Fatalf("stderr = %q, want missing input error", stderr.String())
	}
}

func TestDefaultRunnerResolvesRelativeWorkdir(t *testing.T) {
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	writeCLITestWorkflow(t, workflowsLib)

	root := t.TempDir()
	withWorkingDir(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent",
		"--workflows-lib", workflowsLib,
		"--input", "name=rain",
		"--workdir", "run/output",
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	info, err := os.Stat(filepath.Join(root, "run", "output"))
	if err != nil {
		t.Fatalf("expected relative workdir to be created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("workdir path is not a directory")
	}
}

func TestDefaultRunnerCreatesWorkdir(t *testing.T) {
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	workdir := filepath.Join(t.TempDir(), "run", "nested")
	writeCLITestWorkflow(t, workflowsLib)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent",
		"--workflows-lib", workflowsLib,
		"--input", "name=rain",
		"--workdir", workdir,
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	info, err := os.Stat(workdir)
	if err != nil {
		t.Fatalf("expected workdir to be created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("workdir path is not a directory")
	}
}

func TestDefaultRunnerUsesDefaultWorkflowsLibFromProjectRoot(t *testing.T) {
	projectDir := t.TempDir()
	workflowsLib := filepath.Join(projectDir, ".daiag", "workflows")
	writeCLITestWorkflow(t, workflowsLib)
	cwd := filepath.Join(projectDir, "nested")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", cwd, err)
	}
	withWorkingDir(t, cwd)

	workdir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent",
		"--input", "name=rain",
		"--workdir", workdir,
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "workflow done id=parent status=success") {
		t.Fatalf("stdout missing workflow success:\n%s", stdout.String())
	}
}

func TestDefaultRunnerRejectsPathStyleWorkflowReference(t *testing.T) {
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	if err := os.MkdirAll(workflowsLib, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", workflowsLib, err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "parent.star",
		"--workflows-lib", workflowsLib,
		"--workdir", t.TempDir(),
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), `workflow reference "parent.star" must be a workflow ID`) {
		t.Fatalf("stderr = %q, want workflow ID error", stderr.String())
	}
}

func TestResolveWorkflowsLibResolvesExplicitRelativePath(t *testing.T) {
	root := t.TempDir()
	workflowsLib := filepath.Join(root, "workflows")
	if err := os.MkdirAll(workflowsLib, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", workflowsLib, err)
	}
	withWorkingDir(t, root)

	got, err := resolveWorkflowsLib("workflows")
	if err != nil {
		t.Fatalf("resolveWorkflowsLib() error = %v", err)
	}
	want, err := filepath.Abs("workflows")
	if err != nil {
		t.Fatalf("Abs(workflows): %v", err)
	}
	if got != want {
		t.Fatalf("resolveWorkflowsLib() = %q, want %q", got, want)
	}
}

func TestResolveWorkflowsLibRejectsMissingExplicitPath(t *testing.T) {
	_, err := resolveWorkflowsLib(filepath.Join(t.TempDir(), "missing"))
	if err == nil || !strings.Contains(err.Error(), "--workflows-lib") {
		t.Fatalf("resolveWorkflowsLib() error = %v, want --workflows-lib error", err)
	}
}

func TestResolveWorkflowsLibReportsMissingDefaultProjectRoot(t *testing.T) {
	root := t.TempDir()
	withWorkingDir(t, root)

	_, err := resolveWorkflowsLib("")
	if err == nil || !strings.Contains(err.Error(), "--workflows-lib omitted and no .daiag directory found") {
		t.Fatalf("resolveWorkflowsLib() error = %v, want missing project root error", err)
	}
}

func TestDefaultRunnerValidatesValidWorkflow(t *testing.T) {
	workflowsLib := t.TempDir()
	writeValidateTestWorkflow(t, workflowsLib)

	var stdout, stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"validate",
		"--workflow", "simple",
		"--workflows-lib", workflowsLib,
	})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%q)", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `workflow "simple" is valid`) {
		t.Fatalf("stdout = %q, want success message", stdout.String())
	}
}

func TestDefaultRunnerValidateRejectsMissingInput(t *testing.T) {
	workflowsLib := filepath.Join(t.TempDir(), "workflows")
	writeCLITestWorkflow(t, workflowsLib)

	var stdout, stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"validate",
		"--workflow", "parent",
		"--workflows-lib", workflowsLib,
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), `missing workflow input "name"`) {
		t.Fatalf("stderr = %q, want missing input error", stderr.String())
	}
}

func TestDefaultRunnerValidateRejectsUnknownWorkflow(t *testing.T) {
	workflowsLib := t.TempDir()

	var stdout, stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"validate",
		"--workflow", "unknown",
		"--workflows-lib", workflowsLib,
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "unknown") {
		t.Fatalf("stderr = %q, want unknown workflow error", stderr.String())
	}
}

func writeValidateTestWorkflow(t *testing.T, workflowsLib string) {
	t.Helper()
	writeCLITestFile(t, filepath.Join(workflowsLib, "simple", "simple.star"), `
wf = workflow(id = "simple", steps = [])
`)
}

func writeCLITestWorkflow(t *testing.T, workflowsLib string) string {
	t.Helper()

	parentDir := filepath.Join(workflowsLib, "parent")
	parentPath := filepath.Join(parentDir, "parent.star")
	writeCLITestFile(t, parentPath, `
name = input("name")
spec_path = format("docs/{name}/spec.md", name = name)

wf = workflow(
    id = "parent",
    inputs = ["name"],
    steps = [
        subworkflow(
            id = "child",
            workflow = "child",
            inputs = {
                "name": name,
                "spec_path": spec_path,
            },
        ),
    ],
)
`)
	writeCLITestFile(t, filepath.Join(workflowsLib, "child", "child.star"), `
name = input("name")
spec_path = input("spec_path")

wf = workflow(
    id = "child",
    inputs = ["name", "spec_path"],
    steps = [],
    output_artifacts = {
        "spec": spec_path,
    },
    output_results = {
        "name": name,
    },
)
`)
	return parentPath
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd(): %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("restore working directory %q: %v", oldDir, err)
		}
	})
}

func writeCLITestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
