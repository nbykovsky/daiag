# Workflow Runner V1 Spec

## Purpose

Build a CLI tool in Go that executes prompt-driven workflows defined in Starlark.

The runner is meant for workflows where:

- tasks read and write real files in a workspace
- downstream tasks receive paths to files produced earlier
- each task runs through either the Codex CLI or the Claude Code CLI
- a workflow may repeat a block of steps until a review condition is satisfied

This document defines the first version of that tool.

## Goals

- simple CLI entrypoint
- Starlark workflow definition
- explicit per-task executor selection
- persisted file artifacts on local disk
- JSON result parsing from each task
- sequential execution with human-readable progress output
- one loop primitive: `repeat_until`

## Non-Goals

V1 does not include:

- parallel execution
- branching step types beyond `repeat_until`
- collection loops such as `foreach`
- remote artifact storage
- complex result schemas
- interactive TUI
- workflow resume

## User Experience

The expected usage pattern is:

1. user writes prompt files such as `agents/poem-writer.md`
2. user writes a workflow file such as `workflows/poem.star`
3. user runs the CLI with the workflow path and input params
4. runner evaluates the workflow and executes steps in order
5. runner prints progress to the screen while tasks run
6. workflow-created files remain in the workspace for later steps and for the user

## CLI

### Command

V1 exposes one command:

```sh
daiag run --workflow workflows/poem.star --param name=rain
```

### Flags

- `--workflow <path>`
  Required. Path to the Starlark workflow file.
- `--param <key=value>`
  Optional and repeatable. Supplies workflow parameters used by `param(...)`.
- `--workdir <path>`
  Optional. Working directory for workflow execution. If omitted, use the current directory.
- `--verbose`
  Optional. Prints additional details such as rendered prompt source path and artifact paths.

### Exit Codes

- `0`
  Workflow completed successfully.
- `1`
  Workflow failed during validation or execution.
- `2`
  CLI usage error such as missing required flags or malformed `--param`.

### Path Resolution

- the workflow file path is resolved from the current shell directory
- relative paths inside the workflow are resolved from `--workdir` when provided
- otherwise relative paths are resolved from the current shell directory

## Screen Output

The CLI must print progress to the screen by default.

Output is human-readable line-oriented text, not JSON.
By default, the runner prints its own progress events rather than streaming raw backend chatter.

### Required Events

- workflow start
- step start
- step finish
- loop iteration start
- loop condition result
- workflow finish
- workflow failure

### Example

```text
[12:00:01] workflow start id=poem file=workflows/poem.star
[12:00:01] step start id=write_poem cli=codex model=gpt-5.4
[12:00:08] step done id=write_poem artifacts=poem
[12:00:08] loop iter id=extend_until_ready n=1
[12:00:08] step start id=extend_poem cli=codex model=gpt-5.4
[12:00:15] step done id=extend_poem artifacts=poem
[12:00:15] step start id=review_poem cli=claude model=sonnet
[12:00:20] step done id=review_poem artifacts=review outcome=not_ready
[12:00:20] loop check id=extend_until_ready result=continue
[12:00:21] loop iter id=extend_until_ready n=2
[12:00:42] workflow done id=poem status=success
```

### Logging Rules

- a step start line must include step ID, CLI, and model
- a step done line should include declared artifact keys
- if the result JSON includes an `outcome` field, it may be included in the step done line
- failures should include the failing step ID and a short error message
- raw executor stdout and stderr are captured by the runner and are not streamed directly by default

## Workflow File

### DSL Choice

Use Starlark rather than YAML.

Reasoning:

- step composition is cleaner
- workflow helpers can be written as normal functions
- symbolic references are easier to represent
- loop semantics fit naturally

### Entry Point

The workflow file must define a top-level variable named `wf`.

Example:

```python
wf = workflow(
    id = "poem",
    steps = [...],
)
```

If `wf` is missing, workflow loading fails.

## Core Concepts

### Artifact

An artifact is a file path that a task creates or updates in the workspace.

Example:

```python
artifact("docs/features/rain/poem.md")
```

Artifacts are first-class workflow outputs because downstream tasks consume their paths.

### Result

A result is the JSON object returned by a task on stdout.

V1 requires each task to return a JSON object.
The runner parses that object and checks that required keys are present.

Example:

```json
{
  "topic": "midnight rain",
  "line_count": 4,
  "poem_path": "docs/features/rain/poem.md"
}
```

### Executor

Each task runs through an executor.

Minimal executor shape:

```python
{"cli": "codex", "model": "gpt-5.4"}
```

or:

```python
{"cli": "claude", "model": "sonnet"}
```

`cli` selects the backend adapter.
`model` is a pass-through string for that backend.

### References

Two reference kinds are needed:

- `path_ref(step_id, artifact_key)`
- `json_ref(step_id, field)`

`path_ref` is used when a prompt needs the path to a previously declared artifact.
`json_ref` is used when a workflow condition needs a field from a previous task result.

## DSL Surface

V1 supports these builtins:

- `workflow(...)`
- `task(...)`
- `repeat_until(...)`
- `artifact(path)`
- `path_ref(step_id, artifact_key)`
- `json_ref(step_id, field)`
- `template_file(path, vars = {...})`
- `param(name)`
- `format(template, ...)`
- `eq(left, right)`

### `workflow(...)`

Required fields:

- `id`
- `steps`

Optional fields:

- `default_executor`

Example:

```python
workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [...],
)
```

### `task(...)`

Required fields:

- `id`
- `prompt`
- `artifacts`
- `result_keys`

Optional fields:

- `executor`

Example:

```python
task(
    id = "write_poem",
    prompt = template_file(...),
    artifacts = {
        "poem": artifact("docs/features/rain/poem.md"),
    },
    result_keys = ["topic", "line_count", "poem_path"],
)
```

### `repeat_until(...)`

Required fields:

- `id`
- `steps`
- `until`
- `max_iters`

Example:

```python
repeat_until(
    id = "extend_until_ready",
    max_iters = 8,
    steps = [...],
    until = eq(json_ref("review_poem", "outcome"), "ready"),
)
```

### `artifact(path)`

Declares a file path exposed by the task under some artifact key.

Example:

```python
artifact("docs/features/rain/review.txt")
```

### `path_ref(step_id, artifact_key)`

Returns a symbolic reference to a previously declared artifact path.

Example:

```python
path_ref("write_poem", "poem")
```

### `json_ref(step_id, field)`

Returns a symbolic reference to a field in a previous task result.

Example:

```python
json_ref("review_poem", "outcome")
```

### `template_file(path, vars = {...})`

Loads a prompt template from disk and substitutes variables into it.

Example:

```python
template_file(
    "agents/poem-writer.md",
    vars = {
        "SPEC_PATH": "docs/features/rain/spec.md",
        "POEM_PATH": "docs/features/rain/poem.md",
    },
)
```

### `param(name)`

Reads a value supplied by `--param`.

Example:

```python
name = param("name")
```

If the parameter is missing, workflow loading fails.

### `format(template, ...)`

Formats a string using named placeholders.

Example:

```python
format("docs/features/{name}/poem.md", name = "rain")
```

### `eq(left, right)`

Builds a runtime equality predicate.

This is needed because `json_ref(...)` is symbolic and cannot be compared directly using plain Starlark operators.

Example:

```python
eq(json_ref("review_poem", "outcome"), "ready")
```

## Prompt Templates

Prompt files should be plain text or Markdown files stored in the repository.

They should be path-driven, not name-driven.

Preferred:

```md
Read "${SPEC_PATH}" and write a poem to "${POEM_PATH}".
```

Not preferred:

```md
Read "docs/features/${NAME}/spec.md" and write to "docs/features/${NAME}/poem.md".
```

### Placeholder Rules

- placeholder format is `${NAME}`
- all placeholders referenced in the file must be provided in `vars`
- missing placeholder values cause validation failure before step execution
- values are stringified before substitution

### Prompt Contract

Prompts should instruct the backend to return JSON only.

Minimal example:

```md
# Poem Writer

Read "${SPEC_PATH}" and write a poem to "${POEM_PATH}".

Requirements:

1. Read "${SPEC_PATH}" and extract the topic from the line that starts with `Topic:`.
2. Replace "${POEM_PATH}" with exactly 4 non-empty lines about that topic.
3. Keep the poem as plain text with one line per row and no heading or commentary.
4. Return JSON only with these keys:
   - `topic`
   - `line_count`
   - `poem_path`

Do not wrap the JSON in Markdown fences.
```

## Execution Semantics

### Sequential Execution

All workflow execution is sequential in V1.
Only one task runs at a time.

### Effective Executor

Executor resolution order:

1. task-level `executor`
2. workflow-level `default_executor`
3. fail if neither is present

### Task Lifecycle

For each task, the runner must:

1. resolve the effective executor
2. resolve all references used by the prompt and artifact declarations
3. render the prompt text
4. invoke the selected backend CLI in the workflow workdir
5. wait for task completion
6. parse stdout as a JSON object
7. check that all `result_keys` are present
8. verify that declared artifact paths exist after the task
9. register artifact paths and result fields in runtime state
10. emit a step done event

If any stage fails, workflow execution stops.

### Artifact Behavior

The runner does not write artifact contents itself.
The underlying Codex or Claude task edits files in the workspace.

The artifact declaration means:

- this path is expected to exist after the task
- this path is exposed to downstream tasks by artifact key

Artifacts may point to files that already existed before the task.
In that case the task is allowed to update them.

In V1, artifact declarations are not a filesystem sandbox.
The runner validates declared artifact paths, but it does not attempt to block or diff every extra file change made by the backend.

### Reference Resolution

References resolve against the latest successful execution of the target step.

This is especially important inside `repeat_until`.
If a step runs multiple times, its latest result and latest artifact registration become the current reference target.

History is still kept internally, but downstream resolution uses the latest successful value.

### Loop Semantics

`repeat_until` executes its nested steps in order.

After the last nested step finishes, the runner evaluates the `until` predicate.

- if the predicate is `true`, the loop finishes successfully
- if the predicate is `false`, the next iteration starts
- if the iteration count reaches `max_iters` before success, the workflow fails

### Step IDs

Step IDs must be globally unique across the whole workflow, including steps nested inside loops.

This avoids ambiguous references and simplifies runtime state.

## Executor Backends

The runner should use a small internal adapter interface so the execution engine is independent from CLI details.

Conceptually:

```go
type Executor interface {
    Run(ctx context.Context, req TaskRequest) (TaskResponse, error)
}
```

### TaskRequest

At minimum, a request should contain:

- rendered prompt text
- selected model
- working directory
- task ID
- workflow ID

### TaskResponse

At minimum, a response should contain:

- stdout text
- stderr text if available
- exit status

### Backend Names

V1 supports:

- `codex`
- `claude`

Each adapter is responsible for:

- mapping `model` to the backend CLI's expected flag format
- sending the rendered prompt to the backend
- returning stdout and exit status to the runtime

The workflow engine should not contain backend-specific shell logic.

## Validation Rules

Validation happens before task execution starts.

The runner must reject workflows that violate any of the following:

- workflow file does not exist
- top-level `wf` variable is missing
- workflow ID is empty
- step ID is empty
- duplicate step IDs
- task has no `artifacts`
- task has no `result_keys`
- `repeat_until.max_iters` is less than `1`
- a referenced step ID does not exist
- a referenced artifact key does not exist on the target step
- a workflow param requested by `param(...)` is missing
- a prompt template file does not exist
- a prompt template variable is missing
- no executor can be resolved for a task

The runner should also validate reference ordering.
A task may only refer to steps that are guaranteed to have executed earlier in the same sequential flow.

## Runtime State

The execution engine needs a runtime state object that tracks:

- workflow ID
- run start time
- workdir
- params
- latest artifact values by `step_id.artifact_key`
- latest result values by `step_id.field`
- iteration count for active loops

The engine may also keep per-step history for debugging, but downstream references should use the latest successful values only.

## Example Workflow

```python
name = param("name")
feature_dir = format("docs/features/{name}", name = name)

wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file(
                "agents/poem-writer.md",
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
            max_iters = 8,
            steps = [
                task(
                    id = "extend_poem",
                    prompt = template_file(
                        "agents/poem-extender.md",
                        vars = {
                            "POEM_PATH": path_ref("write_poem", "poem"),
                        },
                    ),
                    artifacts = {
                        "poem": artifact(path_ref("write_poem", "poem")),
                    },
                    result_keys = ["before_line_count", "after_line_count", "poem_path"],
                ),
                task(
                    id = "review_poem",
                    executor = {"cli": "claude", "model": "sonnet"},
                    prompt = template_file(
                        "agents/poem-reviewer.md",
                        vars = {
                            "POEM_PATH": path_ref("extend_poem", "poem"),
                            "REVIEW_PATH": format("{dir}/review.txt", dir = feature_dir),
                        },
                    ),
                    artifacts = {
                        "review": artifact(format("{dir}/review.txt", dir = feature_dir)),
                    },
                    result_keys = ["outcome", "line_count", "review_path"],
                ),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
```

## Suggested Go Package Split

One reasonable package layout:

- `cmd/daiag`
- `internal/cli`
- `internal/workflow`
- `internal/starlarkdsl`
- `internal/runtime`
- `internal/executor/codex`
- `internal/executor/claude`
- `internal/logging`

## V1 Summary

The first version should be a small CLI runtime with these properties:

- one command: `run`
- workflow passed via `--workflow`
- workflow defined in Starlark
- steps run sequentially
- one loop primitive: `repeat_until`
- tasks declare `executor`, `artifacts`, and `result_keys`
- downstream prompts consume prior file paths through `path_ref(...)`
- loop conditions consume prior JSON results through `json_ref(...)`
- progress is printed to the screen while the workflow runs

That is enough to support the poem writer/extender/reviewer class of workflows without over-designing the system.
