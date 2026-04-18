package initcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAvailableWorkflows(t *testing.T) {
	wfs := AvailableWorkflows()
	if len(wfs) == 0 {
		t.Fatal("expected bundled workflows, got none")
	}
	for _, want := range []string{"workflow_bootstrapper", "code_review_pipeline"} {
		if !slices.Contains(wfs, want) {
			t.Errorf("AvailableWorkflows() missing %q", want)
		}
	}
}

func TestInitAllWorkflows(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := Init(Config{ProjectDir: dir}, &out); err != nil {
		t.Fatalf("Init: %v", err)
	}

	daiagDir := filepath.Join(dir, ".daiag")
	for _, sub := range []string{"workflows", "skills", "agents", "runs"} {
		if _, err := os.Stat(filepath.Join(daiagDir, sub)); err != nil {
			t.Errorf(".daiag/%s missing: %v", sub, err)
		}
	}

	for _, wf := range AvailableWorkflows() {
		wfDir := filepath.Join(daiagDir, "workflows", wf)
		if _, err := os.Stat(wfDir); err != nil {
			t.Errorf("workflow %q missing: %v", wf, err)
		}
	}

	wfsMD := filepath.Join(daiagDir, "workflows", "WORKFLOWS.md")
	if _, err := os.Stat(wfsMD); err != nil {
		t.Errorf("WORKFLOWS.md missing: %v", err)
	}
}

func TestInitSubsetWorkflows(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	selected := []string{"workflow_bootstrapper", "code_review_pipeline"}
	if err := Init(Config{ProjectDir: dir, Workflows: selected}, &out); err != nil {
		t.Fatalf("Init: %v", err)
	}

	daiagDir := filepath.Join(dir, ".daiag")
	for _, wf := range selected {
		if _, err := os.Stat(filepath.Join(daiagDir, "workflows", wf)); err != nil {
			t.Errorf("workflow %q missing: %v", wf, err)
		}
	}

	for _, wf := range AvailableWorkflows() {
		if slices.Contains(selected, wf) {
			continue
		}
		wfDir := filepath.Join(daiagDir, "workflows", wf)
		if _, err := os.Stat(wfDir); err == nil {
			t.Errorf("workflow %q should not have been installed", wf)
		}
	}
}

func TestInitInvalidWorkflow(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	err := Init(Config{ProjectDir: dir, Workflows: []string{"nonexistent"}}, &out)
	if err == nil {
		t.Fatal("expected error for invalid workflow, got nil")
	}
	if !strings.Contains(err.Error(), "not a bundled workflow") {
		t.Errorf("error %q should mention 'not a bundled workflow'", err)
	}
}

func TestInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".daiag"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Init(Config{ProjectDir: dir}, &out)
	if err == nil {
		t.Fatal("expected error when .daiag already exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error %q should mention 'already exists'", err)
	}
}

func TestInitForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".daiag"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Init(Config{ProjectDir: dir, Force: true}, &out); err != nil {
		t.Fatalf("Init with --force: %v", err)
	}
}

func TestInitSkillsAgentsAlwaysPresent(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := Init(Config{ProjectDir: dir, Workflows: []string{"workflow_bootstrapper"}}, &out); err != nil {
		t.Fatalf("Init: %v", err)
	}
	daiagDir := filepath.Join(dir, ".daiag")
	for _, sub := range []string{"skills", "agents"} {
		entries, err := os.ReadDir(filepath.Join(daiagDir, sub))
		if err != nil || len(entries) == 0 {
			t.Errorf(".daiag/%s should be populated: %v", sub, err)
		}
	}
}

func TestInitWorkflowsMDAlwaysPresent(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := Init(Config{ProjectDir: dir, Workflows: []string{"workflow_bootstrapper"}}, &out); err != nil {
		t.Fatalf("Init: %v", err)
	}
	wfsMD := filepath.Join(dir, ".daiag", "workflows", "WORKFLOWS.md")
	if _, err := os.Stat(wfsMD); err != nil {
		t.Errorf("WORKFLOWS.md missing even with subset of workflows: %v", err)
	}
}
