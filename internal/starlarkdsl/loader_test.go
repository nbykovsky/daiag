package starlarkdsl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"daiag/internal/workflow"
)

func TestLoaderLoadValidWorkflow(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Read "${SPEC_PATH}" and write to "${POEM_PATH}".
Return JSON only.`)
	writeFile(t, filepath.Join(baseDir, "agents", "reviewer.md"), `Read "${POEM_PATH}" and write to "${REVIEW_PATH}".
Return JSON only.`)

	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
name = param("name")
feature_dir = format("docs/features/{name}", name = name)

wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file(
                "agents/writer.md",
                vars = {
                    "SPEC_PATH": format("{dir}/spec.md", dir = feature_dir),
                    "POEM_PATH": format("{dir}/poem.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "poem": artifact(format("{dir}/poem.md", dir = feature_dir)),
            },
            result_keys = ["topic", "line_count", "poem_path"],
        ),
        repeat_until(
            id = "extend_until_ready",
            max_iters = 3,
            steps = [
                task(
                    id = "review_poem",
                    executor = {"cli": "claude", "model": "sonnet"},
                    prompt = template_file(
                        "agents/reviewer.md",
                        vars = {
                            "POEM_PATH": path_ref("write_poem", "poem"),
                            "REVIEW_PATH": format(
                                "{dir}/review-{iter}.txt",
                                dir = feature_dir,
                                iter = loop_iter("extend_until_ready"),
                            ),
                        },
                    ),
                    artifacts = {
                        "review": artifact(
                            format(
                                "{dir}/review-{iter}.txt",
                                dir = feature_dir,
                                iter = loop_iter("extend_until_ready"),
                            )
                        ),
                    },
                    result_keys = ["outcome", "review_path"],
                ),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
`)

	loader := Loader{
		Params:  map[string]string{"name": "rain"},
		BaseDir: baseDir,
	}

	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if wf.ID != "poem" {
		t.Fatalf("workflow ID = %q, want %q", wf.ID, "poem")
	}
	if wf.DefaultExecutor == nil || wf.DefaultExecutor.CLI != "codex" || wf.DefaultExecutor.Model != "gpt-5.4" {
		t.Fatalf("default executor = %#v, want codex/gpt-5.4", wf.DefaultExecutor)
	}
	if len(wf.Steps) != 2 {
		t.Fatalf("step count = %d, want 2", len(wf.Steps))
	}

	writeTask, ok := wf.Steps[0].(*workflow.Task)
	if !ok {
		t.Fatalf("wf.Steps[0] = %T, want *workflow.Task", wf.Steps[0])
	}
	if got := writeTask.Prompt.TemplatePath; got != "agents/writer.md" {
		t.Fatalf("prompt path = %q, want agents/writer.md", got)
	}

	loop, ok := wf.Steps[1].(*workflow.RepeatUntil)
	if !ok {
		t.Fatalf("wf.Steps[1] = %T, want *workflow.RepeatUntil", wf.Steps[1])
	}
	if loop.MaxIters != 3 {
		t.Fatalf("MaxIters = %d, want 3", loop.MaxIters)
	}
}

func TestLoaderLoadsPoemExampleWorkflow(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd(): %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	workflowPath := filepath.Join(repoRoot, "examples", "poem", "workflows", "poem.star")

	loader := Loader{
		Params:  map[string]string{"name": "rain"},
		BaseDir: repoRoot,
	}

	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if wf.ID != "poem" {
		t.Fatalf("workflow ID = %q, want %q", wf.ID, "poem")
	}
}

func TestLoaderMissingParam(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `name = param("name")
wf = workflow(id = "x", steps = [])`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `missing workflow param "name"`) {
		t.Fatalf("Load() error = %v, want missing param", err)
	}
}

func TestLoaderRejectsForwardReference(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Read "${POEM_PATH}".`)
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "first",
            prompt = template_file("agents/writer.md", vars = {"POEM_PATH": path_ref("second", "poem")}),
            artifacts = {"poem": artifact("out/one.txt")},
            result_keys = ["ok"],
        ),
        task(
            id = "second",
            prompt = "done",
            artifacts = {"poem": artifact("out/two.txt")},
            result_keys = ["ok"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `unknown step "second"`) {
		t.Fatalf("Load() error = %v, want forward reference error", err)
	}
}

func TestLoaderRejectsMissingPromptVariable(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Read "${SPEC_PATH}" and "${POEM_PATH}".`)
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file("agents/writer.md", vars = {"SPEC_PATH": "docs/spec.md"}),
            artifacts = {"poem": artifact("docs/poem.md")},
            result_keys = ["topic"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `missing prompt variable "POEM_PATH"`) {
		t.Fatalf("Load() error = %v, want missing placeholder error", err)
	}
}

func TestLoaderRejectsMissingExecutor(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    steps = [
        task(
            id = "write_poem",
            prompt = "hello",
            artifacts = {"poem": artifact("docs/poem.md")},
            result_keys = ["topic"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `executor is required`) {
		t.Fatalf("Load() error = %v, want missing executor error", err)
	}
}

func TestLoaderRejectsLoopIterOutsideLoopScope(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = "hello",
            artifacts = {
                "poem": artifact(
                    format("docs/poem-{iter}.md", iter = loop_iter("extend_until_ready"))
                ),
            },
            result_keys = ["topic"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `loop "extend_until_ready" is not active in this scope`) {
		t.Fatalf("Load() error = %v, want loop scope error", err)
	}
}

func TestLoaderRejectsDuplicateStepIDs(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "dup",
            prompt = "one",
            artifacts = {"poem": artifact("docs/one.md")},
            result_keys = ["ok"],
        ),
        task(
            id = "dup",
            prompt = "two",
            artifacts = {"poem": artifact("docs/two.md")},
            result_keys = ["ok"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `duplicate step ID "dup"`) {
		t.Fatalf("Load() error = %v, want duplicate step error", err)
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func contains(s, want string) bool {
	return strings.Contains(s, want)
}
