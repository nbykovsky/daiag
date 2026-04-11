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
	workflowPath := writeCLITestWorkflow(t, workdir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", workflowPath,
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

func TestDefaultRunnerLoadsWorkflowRelativeToEntryFileNotWorkdir(t *testing.T) {
	workflowDir := t.TempDir()
	workdir := t.TempDir()
	workflowPath := writeCLITestWorkflow(t, workflowDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", workflowPath,
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
	workflowPath := writeCLITestWorkflow(t, workdir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", workflowPath,
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
	workflowPath := writeCLITestWorkflow(t, workdir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", workflowPath,
		"--workdir", workdir,
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), `missing workflow input "name"`) {
		t.Fatalf("stderr = %q, want missing input error", stderr.String())
	}
}

func TestDefaultRunnerRejectsRelativeWorkdir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", "workflow.star",
		"--workdir", "relative",
	})

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "--workdir must be an absolute path") {
		t.Fatalf("stderr = %q, want absolute workdir error", stderr.String())
	}
}

func TestDefaultRunnerCreatesWorkdir(t *testing.T) {
	workflowDir := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "run", "nested")
	workflowPath := writeCLITestWorkflow(t, workflowDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewDefault(&stdout, &stderr)

	exitCode := app.Run(context.Background(), []string{
		"run",
		"--workflow", workflowPath,
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

func writeCLITestWorkflow(t *testing.T, workdir string) string {
	t.Helper()

	parentPath := filepath.Join(workdir, "parent.star")
	writeCLITestFile(t, parentPath, `
name = input("name")
spec_path = format("docs/{name}/spec.md", name = name)

wf = workflow(
    id = "parent",
    inputs = ["name"],
    steps = [
        subworkflow(
            id = "child",
            workflow = "child.star",
            inputs = {
                "name": name,
                "spec_path": spec_path,
            },
        ),
    ],
)
`)
	writeCLITestFile(t, filepath.Join(workdir, "child.star"), `
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

func writeCLITestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
