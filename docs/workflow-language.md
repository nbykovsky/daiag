# Workflow Language Specification

## Purpose

This document describes the Starlark workflow language implemented by `daiag`.

Workflow files coordinate prompt templates, executor selection, run artifacts,
workflow outputs, and subworkflow composition.

## Entry Files

The CLI workflow ID resolves to:

```text
<workflows-lib>/<workflow-id>/workflow.star
```

The entry file must define a top-level `wf` value created by `workflow(...)`.
Workflow IDs and subworkflow references must match `[A-Za-z0-9_-]+`; path-style
references such as `./wf.star` or `../wf.star` are rejected.

`load(...)` paths are resolved relative to the importing Starlark module and
must stay under the selected `workflows-lib`.

## Supported Builtins

- `workflow(...)`
- `task(...)`
- `repeat_until(...)`
- `when(...)`
- `subworkflow(...)`
- `artifact(path)`
- `path_ref(step_id, artifact_key)`
- `json_ref(step_id, field)`
- `loop_iter(loop_id)`
- `input(name)`
- `run_dir()`
- `projectdir()`
- `template_file(path, vars = {...})`
- `param(name)` for legacy top-level workflows
- `format(template, ...)`
- `eq(left, right)`

`workdir()` is not supported.

## `workflow(...)`

Required fields:

- `id`
- `steps`

Optional fields:

- `inputs`
- `default_executor`
- `output_artifacts`
- `output_results`

Example:

```python
wf = workflow(
    id = "poem",
    inputs = ["topic"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [],
    output_artifacts = {},
    output_results = {},
)
```

Rules:

- `inputs` must contain unique non-empty strings.
- `steps` contains `task(...)`, `repeat_until(...)`, `when(...)`, and `subworkflow(...)` values.
- `output_artifacts` maps public artifact names to string expressions.
- `output_results` maps public result names to value expressions.
- Workflow outputs may reference declared inputs and steps visible at the end of the workflow.

## `task(...)`

Required fields:

- `id`
- `prompt`
- `artifacts`
- `result_keys`

Optional fields:

- `executor`

Example:

```python
report_path = format("{run_dir}/reports/summary.md", run_dir = run_dir())

task(
    id = "write_report",
    prompt = template_file("write_report.md", vars = {
        "REPORT_PATH": report_path,
    }),
    artifacts = {"report": artifact(report_path)},
    result_keys = ["report_path"],
)
```

Rules:

- `prompt` is either an inline string or `template_file(...)`.
- `artifacts` must be non-empty, and every value must be wrapped with `artifact(...)`.
- `result_keys` must be non-empty and unique.
- A task must have an executor either directly or via `workflow(default_executor=...)`.
- The task's executor must return a JSON object containing every declared result key.

## `repeat_until(...)`

Runs child steps until a predicate is true or `max_iters` is reached.

```python
repeat_until(
    id = "review_loop",
    max_iters = 4,
    steps = [
        review_task,
    ],
    until = eq(json_ref("review_task", "outcome"), "approved"),
)
```

`max_iters` must be at least `1`. The condition is evaluated after each full
loop body execution.

## `when(...)`

Runs one branch of steps based on a runtime predicate.

```python
when(
    id = "repair_if_needed",
    condition = eq(json_ref("triage", "outcome"), "needs_repair"),
    steps = [repair_task],
    else_steps = [],
)
```

The `when(...)` node has no artifacts or results. Branch-internal steps are not
visible to later parent steps or workflow outputs.

## `subworkflow(...)`

Runs another workflow from the same `workflows-lib`.

```python
subworkflow(
    id = "spec_refinement",
    workflow = "spec-refinement",
    inputs = {
        "feature_dir": feature_dir,
        "spec_path": spec_path,
    },
)
```

Rules:

- `workflow` is a workflow ID, not a file path.
- Every child workflow input must be bound.
- Parent workflows can reference only the child workflow's declared `output_artifacts` and `output_results`.
- Child workflows cannot see parent internal task IDs.
- `param(...)` is disabled in child workflow files and helper modules loaded by child workflows.

## Expressions

`artifact(...)`, prompt variables, and workflow outputs accept string or value
expressions depending on the field.

String expressions:

- string literal
- `format(...)`
- `path_ref(...)`
- `input(...)`
- `run_dir()`
- `projectdir()`

Value expressions:

- string literal
- integer literal
- `format(...)`
- `path_ref(...)`
- `json_ref(...)`
- `loop_iter(...)`
- `input(...)`
- `run_dir()`
- `projectdir()`

`format(template, ...)` replaces `{name}` placeholders with named arguments.
Every placeholder must have a value.

`path_ref(step_id, artifact_key)` reads an artifact path from an earlier visible
task or subworkflow.

`json_ref(step_id, field)` reads a JSON result value from an earlier visible
task or subworkflow.

`loop_iter(loop_id)` returns the current 1-based iteration number and is valid
only inside that loop.

`eq(left, right)` is the only supported predicate form.

## Inputs

`input(name)` creates a symbolic runtime input reference.

```python
topic = input("topic")

wf = workflow(
    id = "poem_generator",
    inputs = ["topic"],
    steps = [],
)
```

The name must be declared in the current workflow's `inputs` list. Top-level
inputs come from `daiag run --input key=value`; subworkflow inputs come from the
parent `subworkflow(inputs = {...})` map.

Validation checks that every `input(...)` reference is declared. It does not
require concrete values for declared inputs unless a legacy `param(...)` call is
used.

## `run_dir()`

`run_dir()` resolves at execution time to the CLI run directory.

Use it when a prompt must tell an executor where to write a run artifact:

```python
summary_path = format("{run_dir}/author/summary.md", run_dir = run_dir())
```

Subworkflows share the parent run directory.

## `projectdir()`

`projectdir()` resolves at execution time to the CLI project directory.

Use it when a workflow must reference an editable project file:

```python
spec_path = format("{project}/docs/features/login/spec.md", project = projectdir())
```

Executors run with `projectdir` as their current working directory.

## `param(name)`

`param(name)` remains available for legacy top-level workflows. The CLI no
longer has a `--param` flag; top-level `param(...)` reads from the same values
provided with `--input key=value`.

Do not use `param(...)` in new workflows. It is rejected in subworkflow files
and helper modules loaded by subworkflows.

## Prompt Templates

`template_file(path, vars = {...})` loads a Markdown prompt template and
substitutes `${NAME}` placeholders at runtime.

Template paths are resolved relative to the Starlark module where
`template_file(...)` is called.

Every placeholder in the template must have a matching `vars` entry. Prompt
variables are not implicitly resolved against `run-dir`; pass a `run_dir()` or
`path_ref(...)` rooted value when the executor must write a run artifact.

## Artifact Paths

Artifact paths are runtime workflow data, not module paths.

Rules:

- Relative task artifacts and workflow output artifacts resolve under `run-dir`.
- Absolute task artifacts and workflow output artifacts must remain under `projectdir`.
- Task artifacts must exist after the executor finishes and must be files, not directories.
- Existing artifact paths are checked after symlink resolution to prevent escapes from the allowed root.
- Relative paths such as `../outside.md` are rejected.
- Resolved artifact paths are stored as absolute paths and exposed through `path_ref(...)`.

Executors run from `projectdir`, so prompts that ask an executor to write a run
artifact should pass an absolute `run_dir()` path or a path from `path_ref(...)`.

Example:

```python
run_summary = format("{run_dir}/summaries/review.md", run_dir = run_dir())

task(
    id = "write_summary",
    prompt = template_file("write_summary.md", vars = {
        "SUMMARY_PATH": run_summary,
    }),
    artifacts = {"summary": artifact(run_summary)},
    result_keys = ["summary_path"],
)
```

## Workflow Outputs

After all top-level steps complete, the runtime resolves `output_artifacts` and
`output_results` into a `RunResult`.

`daiag run` prints these values after progress messages:

```text
workflow outputs:
artifact summary: /abs/project/.daiag/runs/my_workflow/<run-id>/summaries/review.md
result outcome: "complete"
```

Subworkflow outputs become the artifacts and results of the parent
`subworkflow(...)` step.

## Executors

Executor config is a dict:

```python
{"cli": "codex", "model": "gpt-5.4"}
```

or:

```python
{"cli": "claude", "model": "sonnet"}
```

Codex uses `projectdir` for `-C` and process `Dir`. Claude uses `projectdir`
for process `Dir` and `--add-dir`.

## Validation Errors

Workflow loading and validation reject:

- missing top-level `wf`
- top-level `wf` not created by `workflow(...)`
- invalid workflow IDs and subworkflow references
- missing or invalid loaded modules
- load paths outside `workflows-lib`
- load cycles
- loaded modules that define top-level `wf`
- duplicate or empty workflow inputs
- `input(...)` references not declared in the current workflow
- nil steps, empty step IDs, and duplicate step IDs
- tasks missing prompts, artifacts, result keys, or executors
- duplicate task result keys
- prompt templates with missing variables
- `path_ref(...)` or `json_ref(...)` references to unknown steps or undeclared keys
- `loop_iter(...)` outside its loop scope
- `repeat_until(max_iters < 1)`
- branch cross-references inside `when(...)`
- parent references to branch-internal or child-internal steps
- `param(...)` inside subworkflow files or helper modules loaded by subworkflows

Execution can also fail when:

- a required runtime input is missing
- a task returns no JSON object
- a task result omits a required `result_keys` entry
- a declared task artifact is missing, a directory, or outside the allowed root
- a workflow output cannot be resolved

## Minimal Example

```python
workflow_id = "poem_generator"
n = input("n")
poem_path = format("{run_dir}/{workflow_id}/poem.md", run_dir = run_dir(), workflow_id = workflow_id)

wf = workflow(
    id = workflow_id,
    inputs = ["n"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file("write_poem.md", vars = {
                "N": n,
                "POEM_PATH": poem_path,
            }),
            artifacts = {"poem": artifact(poem_path)},
            result_keys = ["poem_path"],
        ),
    ],
    output_artifacts = {"poem": path_ref("write_poem", "poem")},
    output_results = {"poem_path": json_ref("write_poem", "poem_path")},
)
```
