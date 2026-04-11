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
default_executor = {"cli": "codex", "model": "gpt-5.4"}

poem_path = format("{dir}/poem.md", dir = feature_dir)
review_path = format(
    "{dir}/review-{iter}.txt",
    dir = feature_dir,
    iter = loop_iter("extend_until_ready"),
)

write_poem = task(
    id = "write_poem",
    prompt = template_file(
        "agents/writer.md",
        vars = {
            "SPEC_PATH": format("{dir}/spec.md", dir = feature_dir),
            "POEM_PATH": poem_path,
        },
    ),
    artifacts = {
        "poem": artifact(poem_path),
    },
    result_keys = ["topic", "line_count", "poem_path"],
)

review_poem = task(
    id = "review_poem",
    executor = {"cli": "claude", "model": "sonnet"},
    prompt = template_file(
        "agents/reviewer.md",
        vars = {
            "POEM_PATH": path_ref("write_poem", "poem"),
            "REVIEW_PATH": review_path,
        },
    ),
    artifacts = {
        "review": artifact(review_path),
    },
    result_keys = ["outcome", "review_path"],
)

wf = workflow(
    id = "poem",
    default_executor = default_executor,
    steps = [
        write_poem,
        repeat_until(
            id = "extend_until_ready",
            max_iters = 3,
            steps = [
                review_poem,
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

func TestLoaderLoadsWorkflowWithModules(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Read "${POEM_PATH}".`)
	writeFile(t, filepath.Join(baseDir, "lib", "common.star"), `
def poem_result_keys():
    return ["topic"]
`)
	writeFile(t, filepath.Join(baseDir, "lib", "tasks.star"), `
load("common.star", "poem_result_keys")

default_executor = {"cli": "codex", "model": "gpt-5.4"}

def write_poem_task(poem_path):
    return task(
        id = "write_poem",
        prompt = template_file("../agents/writer.md", vars = {"POEM_PATH": poem_path}),
        artifacts = {"poem": artifact(poem_path)},
        result_keys = poem_result_keys(),
    )
`)
	writeFile(t, workflowPath, `
load("lib/tasks.star", "default_executor", "write_poem_task")

wf = workflow(
    id = "poem",
    default_executor = default_executor,
    steps = [
        write_poem_task("docs/poem.md"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if wf.ID != "poem" {
		t.Fatalf("workflow ID = %q, want %q", wf.ID, "poem")
	}
	if len(wf.Steps) != 1 {
		t.Fatalf("step count = %d, want 1", len(wf.Steps))
	}
	task, ok := wf.Steps[0].(*workflow.Task)
	if !ok {
		t.Fatalf("wf.Steps[0] = %T, want *workflow.Task", wf.Steps[0])
	}
	if got := task.Prompt.TemplateDir; got != filepath.Join(baseDir, "lib") {
		t.Fatalf("TemplateDir = %q, want %q", got, filepath.Join(baseDir, "lib"))
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

func TestLoaderLoadsDevelopmentWorkflowExample(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd(): %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	workflowPath := filepath.Join(repoRoot, "examples", "development-workflow", "workflows", "feature-development.star")

	loader := Loader{
		Inputs:  map[string]string{"name": "indicators"},
		BaseDir: repoRoot,
	}

	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if wf.ID != "feature-development" {
		t.Fatalf("workflow ID = %q, want %q", wf.ID, "feature-development")
	}
	sub, ok := wf.Steps[0].(*workflow.Subworkflow)
	if !ok {
		t.Fatalf("wf.Steps[0] = %T, want *workflow.Subworkflow", wf.Steps[0])
	}
	if sub.Workflow == nil || sub.Workflow.ID != "spec-refinement" {
		t.Fatalf("subworkflow child = %#v, want spec-refinement", sub.Workflow)
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

func TestLoaderLoadsWorkflowInputs(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Read "${SPEC_PATH}" and write "${POEM_PATH}".`)
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
spec_path = input("spec_path")
feature_dir = input("feature_dir")
poem_path = format("{dir}/poem.md", dir = feature_dir)

wf = workflow(
    id = "poem",
    inputs = ["spec_path", "feature_dir"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file(
                "agents/writer.md",
                vars = {
                    "SPEC_PATH": spec_path,
                    "POEM_PATH": poem_path,
                },
            ),
            artifacts = {"poem": artifact(poem_path)},
            result_keys = ["ok"],
        ),
    ],
)
`)

	loader := Loader{
		Inputs: map[string]string{
			"spec_path":   "docs/spec.md",
			"feature_dir": "docs/features/rain",
		},
		BaseDir: baseDir,
	}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := strings.Join(wf.Inputs, ","); got != "spec_path,feature_dir" {
		t.Fatalf("Inputs = %q, want spec_path,feature_dir", got)
	}
}

func TestLoaderLoadsWorkdirExpression(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Write "${POEM_PATH}".`)
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
poem_path = format("{workdir}/docs/poem.md", workdir = workdir())

wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file("agents/writer.md", vars = {"POEM_PATH": poem_path}),
            artifacts = {"poem": artifact(poem_path)},
            result_keys = ["ok"],
        ),
    ],
    output_artifacts = {"poem": poem_path},
)
`)

	loader := Loader{BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	task := wf.Steps[0].(*workflow.Task)
	expr, ok := task.Artifacts["poem"].(workflow.FormatExpr)
	if !ok {
		t.Fatalf("artifact poem = %T, want workflow.FormatExpr", task.Artifacts["poem"])
	}
	if _, ok := expr.Args["workdir"].(workflow.WorkdirRef); !ok {
		t.Fatalf("format workdir arg = %T, want workflow.WorkdirRef", expr.Args["workdir"])
	}
	if _, ok := wf.OutputArtifacts["poem"].(workflow.FormatExpr); !ok {
		t.Fatalf("output artifact poem = %T, want workflow.FormatExpr", wf.OutputArtifacts["poem"])
	}
}

func TestLoaderLoadsProjectdirFromCallingModule(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".daiag"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.daiag): %v", err)
	}
	workflowDir := filepath.Join(projectDir, ".daiag", "workflows")
	writeFile(t, filepath.Join(workflowDir, "agents", "writer.md"), `Write "${POEM_PATH}".`)
	writeFile(t, filepath.Join(workflowDir, "lib", "paths.star"), `
project_root = projectdir()

def poem_path():
    return format("{project}/docs/poem.md", project = project_root)
`)
	workflowPath := filepath.Join(workflowDir, "workflow.star")
	writeFile(t, workflowPath, `
load("lib/paths.star", "poem_path")

wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file("agents/writer.md", vars = {"POEM_PATH": poem_path()}),
            artifacts = {"poem": artifact(poem_path())},
            result_keys = ["ok"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: workflowDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	task := wf.Steps[0].(*workflow.Task)
	expr, ok := task.Artifacts["poem"].(workflow.FormatExpr)
	if !ok {
		t.Fatalf("artifact poem = %T, want workflow.FormatExpr", task.Artifacts["poem"])
	}
	project, ok := expr.Args["project"].(workflow.Literal)
	if !ok {
		t.Fatalf("format project arg = %T, want workflow.Literal", expr.Args["project"])
	}
	if project.Value != projectDir {
		t.Fatalf("projectdir() = %q, want %q", project.Value, projectDir)
	}
}

func TestLoaderRejectsProjectdirOutsideProject(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
project_root = projectdir()

wf = workflow(id = "bad", steps = [])
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `no .daiag directory found`) {
		t.Fatalf("Load() error = %v, want projectdir error", err)
	}
	if !contains(err.Error(), `pass the project path as an explicit workflow input instead`) {
		t.Fatalf("Load() error = %v, want explicit input suggestion", err)
	}
}

func TestLoaderRejectsDuplicateWorkflowInputs(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    inputs = ["name", "name"],
    steps = [],
)
`)

	loader := Loader{Inputs: map[string]string{"name": "rain"}, BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `workflow inputs must be unique`) {
		t.Fatalf("Load() error = %v, want duplicate input error", err)
	}
}

func TestLoaderRejectsUndeclaredInput(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
name = input("name")
wf = workflow(
    id = "bad",
    inputs = ["other"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = "hello",
            artifacts = {"poem": artifact(name)},
            result_keys = ["ok"],
        ),
    ],
)
`)

	loader := Loader{Inputs: map[string]string{"other": "rain"}, BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `unknown workflow input "name"`) {
		t.Fatalf("Load() error = %v, want undeclared input error", err)
	}
}

func TestLoaderRejectsMissingWorkflowInput(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    inputs = ["name"],
    steps = [],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `missing workflow input "name"`) {
		t.Fatalf("Load() error = %v, want missing input error", err)
	}
}

func TestLoaderLoadsWorkflowOutputContracts(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = "hello",
            artifacts = {"poem": artifact("docs/poem.md")},
            result_keys = ["ok"],
        ),
    ],
    output_artifacts = {"poem": path_ref("write_poem", "poem")},
    output_results = {"ok": json_ref("write_poem", "ok")},
)
`)

	loader := Loader{BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(wf.OutputArtifacts) != 1 {
		t.Fatalf("OutputArtifacts len = %d, want 1", len(wf.OutputArtifacts))
	}
	if len(wf.OutputResults) != 1 {
		t.Fatalf("OutputResults len = %d, want 1", len(wf.OutputResults))
	}
}

func TestLoaderLoadsWorkflowOutputArtifactFromInput(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
spec_path = input("spec_path")

wf = workflow(
    id = "spec",
    inputs = ["spec_path"],
    steps = [],
    output_artifacts = {"spec": spec_path},
)
`)

	loader := Loader{Inputs: map[string]string{"spec_path": "docs/spec.md"}, BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, ok := wf.OutputArtifacts["spec"].(workflow.InputRef); !ok {
		t.Fatalf("OutputArtifacts[spec] = %T, want workflow.InputRef", wf.OutputArtifacts["spec"])
	}
}

func TestLoaderRejectsOutputReferenceToUnknownStep(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "bad",
    steps = [],
    output_artifacts = {"poem": path_ref("missing", "poem")},
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `unknown step "missing"`) {
		t.Fatalf("Load() error = %v, want unknown step error", err)
	}
}

func TestLoaderRejectsOutputReferenceToUnknownArtifact(t *testing.T) {
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
            artifacts = {"poem": artifact("docs/poem.md")},
            result_keys = ["ok"],
        ),
    ],
    output_artifacts = {"review": path_ref("write_poem", "review")},
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `step "write_poem" does not declare artifact "review"`) {
		t.Fatalf("Load() error = %v, want unknown artifact error", err)
	}
}

func TestLoaderRejectsOutputReferenceToUnknownResult(t *testing.T) {
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
            artifacts = {"poem": artifact("docs/poem.md")},
            result_keys = ["ok"],
        ),
    ],
    output_results = {"missing": json_ref("write_poem", "missing")},
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `step "write_poem" does not declare result key "missing"`) {
		t.Fatalf("Load() error = %v, want unknown result error", err)
	}
}

func TestLoaderAcceptsOutputReferenceToLoopBodyTask(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        repeat_until(
            id = "review_until_ready",
            max_iters = 1,
            steps = [
                task(
                    id = "review_poem",
                    prompt = "hello",
                    artifacts = {"review": artifact("docs/review.md")},
                    result_keys = ["outcome"],
                ),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
    output_artifacts = {"review": path_ref("review_poem", "review")},
    output_results = {"outcome": json_ref("review_poem", "outcome")},
)
`)

	loader := Loader{BaseDir: baseDir}
	if _, err := loader.Load(workflowPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoaderLoadsSubworkflowNode(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	childPath := filepath.Join(baseDir, "child.star")
	writeFile(t, childPath, `
wf = workflow(
    id = "child",
    steps = [],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "spec", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	sub, ok := wf.Steps[0].(*workflow.Subworkflow)
	if !ok {
		t.Fatalf("wf.Steps[0] = %T, want *workflow.Subworkflow", wf.Steps[0])
	}
	if sub.Workflow == nil || sub.Workflow.ID != "child" {
		t.Fatalf("sub.Workflow = %#v, want child workflow", sub.Workflow)
	}
	if sub.WorkflowPath != childPath {
		t.Fatalf("sub.WorkflowPath = %q, want %q", sub.WorkflowPath, childPath)
	}
	if len(sub.Inputs) != 0 {
		t.Fatalf("sub.Inputs len = %d, want 0", len(sub.Inputs))
	}
}

func TestLoaderResolvesSubworkflowPathRelativeToCallerModule(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	childPath := filepath.Join(baseDir, "children", "spec.star")
	writeFile(t, childPath, `
wf = workflow(
    id = "spec",
    steps = [],
)
`)
	writeFile(t, filepath.Join(baseDir, "lib", "stages.star"), `
def spec_stage():
    return subworkflow(id = "spec", workflow = "../children/spec.star")
`)
	writeFile(t, workflowPath, `
load("lib/stages.star", "spec_stage")

wf = workflow(
    id = "parent",
    steps = [
        spec_stage(),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	sub := wf.Steps[0].(*workflow.Subworkflow)
	if sub.WorkflowPath != childPath {
		t.Fatalf("WorkflowPath = %q, want %q", sub.WorkflowPath, childPath)
	}
	if sub.ModuleDir != filepath.Join(baseDir, "lib") {
		t.Fatalf("ModuleDir = %q, want %q", sub.ModuleDir, filepath.Join(baseDir, "lib"))
	}
}

func TestLoaderRejectsSubworkflowOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	outsidePath := filepath.Join(filepath.Dir(baseDir), "outside.star")
	writeFile(t, outsidePath, `
wf = workflow(id = "outside", steps = [])
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "outside", workflow = "../outside.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `escapes base directory`) {
		t.Fatalf("Load() error = %v, want base directory error", err)
	}
}

func TestLoaderRejectsSubworkflowWithoutWF(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `value = "no wf"`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `does not define top-level wf`) {
		t.Fatalf("Load() error = %v, want missing wf error", err)
	}
}

func TestLoaderRejectsParamInSubworkflow(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
name = param("name")

wf = workflow(id = "child", steps = [])
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{Params: map[string]string{"name": "rain"}, BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `param("name") is not allowed in subworkflows`) {
		t.Fatalf("Load() error = %v, want subworkflow param error", err)
	}
}

func TestLoaderRejectsParamInSubworkflowHelperModule(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "lib", "helper.star"), `
name = param("name")
`)
	writeFile(t, filepath.Join(baseDir, "child.star"), `
load("lib/helper.star", "name")

wf = workflow(id = "child", steps = [])
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{Params: map[string]string{"name": "rain"}, BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `param("name") is not allowed in subworkflows`) {
		t.Fatalf("Load() error = %v, want subworkflow helper param error", err)
	}
}

func TestLoaderRejectsDirectSubworkflowCycle(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "self", workflow = "workflow.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `subworkflow cycle detected`) {
		t.Fatalf("Load() error = %v, want cycle error", err)
	}
}

func TestLoaderRejectsIndirectSubworkflowCycle(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    steps = [
        subworkflow(id = "parent", workflow = "workflow.star"),
    ],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `subworkflow cycle detected`) {
		t.Fatalf("Load() error = %v, want cycle error", err)
	}
}

func TestLoaderAcceptsNoInputSubworkflowWithEmptyInputs(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    steps = [],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "implicit", workflow = "child.star"),
        subworkflow(id = "explicit", workflow = "child.star", inputs = {}),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, node := range wf.Steps {
		sub := node.(*workflow.Subworkflow)
		if len(sub.Inputs) != 0 {
			t.Fatalf("subworkflow %q inputs len = %d, want 0", sub.ID, len(sub.Inputs))
		}
	}
}

func TestLoaderLoadsDistinctSubworkflowInstancesForSameFile(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    steps = [],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "first", workflow = "child.star"),
        subworkflow(id = "second", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	wf, err := loader.Load(workflowPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	first := wf.Steps[0].(*workflow.Subworkflow)
	second := wf.Steps[1].(*workflow.Subworkflow)
	if first.Workflow == second.Workflow {
		t.Fatal("subworkflow instances share the same child workflow pointer")
	}
}

func TestLoaderAllowsParentReferencesToSubworkflowOutputs(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_spec",
            prompt = "hello",
            artifacts = {"spec": artifact("docs/spec.md")},
            result_keys = ["status"],
        ),
    ],
    output_artifacts = {"spec": path_ref("write_spec", "spec")},
    output_results = {"status": json_ref("write_spec", "status")},
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        subworkflow(id = "spec", workflow = "child.star"),
        task(
            id = "consume_spec",
            prompt = "hello",
            artifacts = {
                "spec": artifact(path_ref("spec", "spec")),
                "status": artifact(format("docs/{status}.md", status = json_ref("spec", "status"))),
            },
            result_keys = ["ok"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	if _, err := loader.Load(workflowPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoaderRejectsParentReferenceToChildInternalTask(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_spec",
            prompt = "hello",
            artifacts = {"spec": artifact("docs/spec.md")},
            result_keys = ["status"],
        ),
    ],
    output_artifacts = {"spec": path_ref("write_spec", "spec")},
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        subworkflow(id = "spec", workflow = "child.star"),
        task(
            id = "consume_spec",
            prompt = "hello",
            artifacts = {"spec": artifact(path_ref("write_spec", "spec"))},
            result_keys = ["ok"],
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `unknown step "write_spec"`) {
		t.Fatalf("Load() error = %v, want child internal reference error", err)
	}
}

func TestLoaderRejectsChildReferenceToParentTask(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "consume_parent",
            prompt = "hello",
            artifacts = {"spec": artifact(path_ref("prepare", "spec"))},
            result_keys = ["ok"],
        ),
    ],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "prepare",
            prompt = "hello",
            artifacts = {"spec": artifact("docs/spec.md")},
            result_keys = ["ok"],
        ),
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `unknown step "prepare"`) {
		t.Fatalf("Load() error = %v, want parent reference error", err)
	}
}

func TestLoaderAllowsSameTaskIDAcrossParentAndChild(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_spec",
            prompt = "child",
            artifacts = {"spec": artifact("docs/child-spec.md")},
            result_keys = ["ok"],
        ),
    ],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_spec",
            prompt = "parent",
            artifacts = {"spec": artifact("docs/parent-spec.md")},
            result_keys = ["ok"],
        ),
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	if _, err := loader.Load(workflowPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoaderRejectsDuplicateIDsInsideChildWorkflow(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "dup",
            prompt = "one",
            artifacts = {"spec": artifact("docs/one.md")},
            result_keys = ["ok"],
        ),
        task(
            id = "dup",
            prompt = "two",
            artifacts = {"spec": artifact("docs/two.md")},
            result_keys = ["ok"],
        ),
    ],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `duplicate step ID "dup"`) {
		t.Fatalf("Load() error = %v, want child duplicate ID error", err)
	}
}

func TestLoaderAllowsLoopIterInSubworkflowInputBinding(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    inputs = ["iter"],
    steps = [],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        repeat_until(
            id = "review_loop",
            max_iters = 1,
            steps = [
                subworkflow(
                    id = "child",
                    workflow = "child.star",
                    inputs = {"iter": loop_iter("review_loop")},
                ),
            ],
            until = eq(loop_iter("review_loop"), 1),
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	if _, err := loader.Load(workflowPath); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoaderDoesNotInheritParentLoopScopeInChildWorkflow(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_review",
            prompt = "hello",
            artifacts = {
                "review": artifact(format("docs/review-{iter}.md", iter = loop_iter("review_loop"))),
            },
            result_keys = ["ok"],
        ),
    ],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        repeat_until(
            id = "review_loop",
            max_iters = 1,
            steps = [
                subworkflow(id = "child", workflow = "child.star"),
            ],
            until = eq(loop_iter("review_loop"), 1),
        ),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `loop "review_loop" is not active in this scope`) {
		t.Fatalf("Load() error = %v, want child loop scope error", err)
	}
}

func TestLoaderRejectsMissingSubworkflowInputBinding(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    inputs = ["name"],
    steps = [],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "child", workflow = "child.star"),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `missing input binding "name"`) {
		t.Fatalf("Load() error = %v, want missing binding error", err)
	}
}

func TestLoaderRejectsUnknownSubworkflowInputBinding(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "child.star"), `
wf = workflow(
    id = "child",
    steps = [],
)
`)
	writeFile(t, workflowPath, `
wf = workflow(
    id = "parent",
    steps = [
        subworkflow(id = "child", workflow = "child.star", inputs = {"name": "rain"}),
    ],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `unknown input binding "name"`) {
		t.Fatalf("Load() error = %v, want unknown binding error", err)
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

func TestLoaderRejectsLoadOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	parentFile := filepath.Join(filepath.Dir(baseDir), "outside.star")
	writeFile(t, parentFile, `value = "nope"`)
	writeFile(t, workflowPath, `
load("../outside.star", "value")

wf = workflow(
    id = "bad",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `escapes base directory`) {
		t.Fatalf("Load() error = %v, want base directory error", err)
	}
}

func TestLoaderRejectsLoadCycle(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "lib", "a.star"), `
load("b.star", "value_b")

value_a = "a"
`)
	writeFile(t, filepath.Join(baseDir, "lib", "b.star"), `
load("a.star", "value_a")

value_b = "b"
`)
	writeFile(t, workflowPath, `
load("lib/a.star", "value_a")

wf = workflow(
    id = "bad",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `load cycle detected`) {
		t.Fatalf("Load() error = %v, want load cycle error", err)
	}
}

func TestLoaderRejectsWFInLoadedModule(t *testing.T) {
	baseDir := t.TempDir()
	workflowPath := filepath.Join(baseDir, "workflow.star")
	writeFile(t, filepath.Join(baseDir, "lib", "helper.star"), `
wf = workflow(
    id = "nested",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [],
)

helper_value = "x"
`)
	writeFile(t, workflowPath, `
load("lib/helper.star", "helper_value")

wf = workflow(
    id = "bad",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [],
)
`)

	loader := Loader{BaseDir: baseDir}
	_, err := loader.Load(workflowPath)
	if err == nil || !contains(err.Error(), `must not define top-level wf`) {
		t.Fatalf("Load() error = %v, want loaded wf error", err)
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
