package starlarkdsl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorkflowID(t *testing.T) {
	workflowsLib := t.TempDir()
	workflowPath := filepath.Join(workflowsLib, "write_poem", "write_poem.star")
	writeWorkflowRefTestFile(t, workflowPath, "")

	got, err := ResolveWorkflowID(workflowsLib, "write_poem")
	if err != nil {
		t.Fatalf("ResolveWorkflowID() error = %v", err)
	}
	if got != workflowPath {
		t.Fatalf("ResolveWorkflowID() = %q, want %q", got, workflowPath)
	}
}

func TestResolveWorkflowIDRejectsPath(t *testing.T) {
	_, err := ResolveWorkflowID(t.TempDir(), "./write_poem.star")
	if err == nil || !strings.Contains(err.Error(), `workflow reference "./write_poem.star" must be a workflow ID`) {
		t.Fatalf("ResolveWorkflowID() error = %v, want workflow ID error", err)
	}
}

func TestResolveWorkflowIDReportsMissingFile(t *testing.T) {
	workflowsLib := t.TempDir()
	wantPath := filepath.Join(workflowsLib, "write_poem", "write_poem.star")

	_, err := ResolveWorkflowID(workflowsLib, "write_poem")
	if err == nil || !strings.Contains(err.Error(), wantPath) {
		t.Fatalf("ResolveWorkflowID() error = %v, want expected path %q", err, wantPath)
	}
}

func writeWorkflowRefTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
