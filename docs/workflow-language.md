# Workflow Language Specification

## Purpose

This document defines the Starlark-based workflow language supported by `daiag`.

It is the authoring reference for writing valid workflow files.
It covers:

- file structure
- supported builtins
- module loading
- path resolution rules
- execution semantics
- validation rules
- current validation commands

This document describes the implementation that exists today.

## File Type

Workflow files use the `.star` extension and are evaluated as Starlark.

Example:

```python
wf = workflow(
    id = "poem",
    steps = [],
)
```

## Entry File

The file passed to the CLI is the entry workflow file.

Example:

```sh
daiag run --workflow examples/poem/workflows/poem.star --param name=rain
```

The entry file must define a top-level variable named `wf`.

Rules:

- `wf` must exist in the entry file
- `wf` must be created by `workflow(...)`
- loaded helper modules must not define top-level `wf`

## Language Model

The language is ordinary Starlark plus a small set of predeclared DSL builtins.

This means workflows may use normal Starlark features such as:

- variables
- lists
- dicts
- functions
- `load(...)`

Example:

```python
name = param("name")
feature_dir = format("examples/poem/docs/features/{name}", name = name)

write_poem = task(
    id = "write_poem",
    prompt = "hello",
    artifacts = {"poem": artifact(format("{dir}/poem.md", dir = feature_dir))},
    result_keys = ["ok"],
)
```

## Supported Builtins

The workflow DSL provides these builtins:

- `workflow(...)`
- `task(...)`
- `repeat_until(...)`
- `artifact(path)`
- `path_ref(step_id, artifact_key)`
- `json_ref(step_id, field)`
- `loop_iter(loop_id)`
- `template_file(path, vars = {...})`
- `param(name)`
- `format(template, ...)`
- `eq(left, right)`

## `workflow(...)`

Creates the top-level workflow object.

Required fields:

- `id`
- `steps`

Optional fields:

- `default_executor`

Example:

```python
wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [],
)
```

Rules:

- `id` must be non-empty
- `steps` must be a list of `task(...)` and `repeat_until(...)` values

## `task(...)`

Creates one executable step.

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
    prompt = template_file(
        "../agents/poem-writer.md",
        vars = {
            "SPEC_PATH": "examples/poem/docs/features/rain/spec.md",
            "POEM_PATH": "examples/poem/docs/features/rain/poem.md",
        },
    ),
    artifacts = {
        "poem": artifact("examples/poem/docs/features/rain/poem.md"),
    },
    result_keys = ["topic", "line_count", "poem_path"],
)
```

Rules:

- `id` must be non-empty
- `prompt` must be either a string or `template_file(...)`
- `artifacts` must be a non-empty dict
- every artifact value must be declared with `artifact(...)`
- `result_keys` must be a non-empty list of unique non-empty strings
- `executor` must be present either on the task or through `workflow(default_executor=...)`

## `repeat_until(...)`

Creates a loop that executes its body until a predicate becomes true or `max_iters` is reached.

Required fields:

- `id`
- `steps`
- `until`
- `max_iters`

Example:

```python
repeat_until(
    id = "extend_until_ready",
    max_iters = 4,
    steps = [
        extend_poem_task(paths),
        review_poem_task(paths),
    ],
    until = eq(json_ref("review_poem", "outcome"), "ready"),
)
```

Rules:

- `id` must be non-empty
- `max_iters` must be at least `1`
- `steps` must be a list of `task(...)` and `repeat_until(...)`
- `until` must be a supported predicate

Semantics:

- the loop body runs in order
- after the body finishes, the predicate is evaluated
- if the predicate is true, the loop stops
- if the predicate never becomes true, execution fails after `max_iters`

## `artifact(path)`

Declares a task artifact path.

Example:

```python
artifact("examples/poem/docs/features/rain/poem.md")
```

The argument must resolve to a string expression.

Supported string expression types:

- string literal
- `format(...)`
- `path_ref(...)`

## `path_ref(step_id, artifact_key)`

Refers to an artifact path produced by an earlier task.

Example:

```python
path_ref("write_poem", "poem")
```

Rules:

- `step_id` must refer to an earlier step
- the referenced step must declare the artifact key
- forward references are not allowed

## `json_ref(step_id, field)`

Refers to a JSON result field produced by an earlier task.

Example:

```python
json_ref("review_poem", "outcome")
```

Rules:

- `step_id` must refer to an earlier step
- the referenced step must declare the result key in `result_keys`
- forward references are not allowed

## `loop_iter(loop_id)`

Returns the current `repeat_until` iteration number for the named loop.

Example:

```python
loop_iter("extend_until_ready")
```

Rules:

- `loop_id` must be non-empty
- the named loop must structurally enclose the task where it is used

Semantics:

- iteration numbers are `1`-based
- the value exists only while the loop is active

Typical use:

```python
format(
    "{dir}/review-{iter}.txt",
    dir = feature_dir,
    iter = loop_iter("extend_until_ready"),
)
```

## `template_file(path, vars = {...})`

Loads a prompt template from disk and substitutes `${NAME}` placeholders at runtime.

Example:

```python
template_file(
    "../agents/poem-reviewer.md",
    vars = {
        "POEM_PATH": path_ref("extend_poem", "poem"),
        "REVIEW_PATH": paths["review_path"],
    },
)
```

Rules:

- `path` must be a string
- `vars` must be a dict when provided
- every template placeholder must have a corresponding entry in `vars`

Path resolution rule:

- `template_file(...)` paths are resolved relative to the Starlark module where the call appears

This matters when using `load(...)`.

Example:

- module file: `examples/poem/workflows/lib/tasks.star`
- prompt path: `../../agents/poem-writer.md`
- resolved prompt file: `examples/poem/agents/poem-writer.md`

## `param(name)`

Reads a workflow parameter supplied on the CLI.

Example:

```python
name = param("name")
```

CLI:

```sh
daiag run --workflow examples/poem/workflows/poem.star --param name=rain
```

Rules:

- the named parameter must be supplied
- missing parameters are a workflow-load error

Current implementation note:

- `param(...)` works in both entry files and loaded modules
- for readability, it is still recommended to read parameters in the entry file and pass values into helper functions

## `format(template, ...)`

Builds a string from named placeholders in `{name}` form.

Example:

```python
format("{dir}/poem.md", dir = feature_dir)
```

Supported argument value types:

- string literal
- integer literal
- `format(...)`
- `path_ref(...)`
- `json_ref(...)`
- `loop_iter(...)`

Rules:

- the template string must be non-empty
- every `{name}` in the template must have a corresponding keyword argument

## `eq(left, right)`

Builds a predicate used by `repeat_until(...)`.

Example:

```python
eq(json_ref("review_poem", "outcome"), "ready")
```

Current implementation supports only equality predicates.

## Executors

Executors are configured as dicts with these keys:

- `cli`
- `model`

Example:

```python
{"cli": "codex", "model": "gpt-5.4"}
```

or:

```python
{"cli": "claude", "model": "sonnet"}
```

Rules:

- both keys must be present when an executor is declared
- a task uses its own executor when present
- otherwise it uses `workflow(default_executor = ...)`

## Results

Each task must return a JSON object on stdout.

Example:

```json
{
  "outcome": "ready",
  "line_count": 6,
  "review_path": "examples/poem/docs/features/rain/review-2.txt"
}
```

Rules:

- the result must be a JSON object
- every key declared in `result_keys` must be present in the returned JSON

Implementation detail:

- the runner accepts raw JSON output
- it also tolerates mixed executor output when a valid JSON object appears inside surrounding text
- validation still applies to the extracted JSON object

## Module Loading

The language supports standard Starlark `load(...)`.

Example:

```python
load("lib/paths.star", "feature_paths")
load("lib/tasks.star", "default_executor", "write_poem_task")
```

Rules:

- load paths must be local filesystem paths
- load paths must end with `.star`
- load paths are resolved relative to the importing module
- load paths must remain under the workflow base directory
- URLs are not allowed
- load cycles are rejected

Loaded modules may export:

- constants
- dicts and lists
- helper functions
- prebuilt `task(...)` values
- prebuilt `repeat_until(...)` values

Recommendation:

- prefer helper functions that build tasks instead of reusing one prebuilt task value multiple times

Reason:

- step IDs must remain globally unique

## Step ID Rules

Step IDs are global across the entire workflow, including nested loops.

This means:

- two tasks may not share the same `id`
- a loop may not reuse a task ID from elsewhere in the workflow

## Artifact Path Rules

Artifact paths are runtime workflow data, not module paths.

This means:

- they are not resolved relative to the Starlark module file
- they are interpreted relative to workflow execution context and `--workdir`
- absolute artifact paths are preserved as-is

## Validation Rules

Workflow loading and validation reject the following cases.

### Entry File Errors

- missing top-level `wf`
- top-level `wf` is not a `workflow(...)`
- missing CLI parameter required by `param(...)`

### Module Errors

- missing loaded file
- invalid load path
- load path outside the workflow base directory
- load path without `.star` suffix
- load cycle
- loaded module defines top-level `wf`
- imported symbol not exported by the loaded module

### Workflow Structure Errors

- empty workflow ID
- nil step
- empty step ID
- duplicate step ID
- unsupported node type

### Task Errors

- missing prompt
- missing artifacts
- missing result keys
- empty artifact key
- empty result key
- duplicate result key
- missing executor
- executor without both `cli` and `model`

### Template Errors

- prompt template file cannot be read
- template placeholder missing from `vars`
- unresolved template placeholder at render time

### Reference Errors

- `path_ref(...)` to unknown step
- `path_ref(...)` to undeclared artifact key
- `json_ref(...)` to unknown step
- `json_ref(...)` to undeclared result key
- `loop_iter(...)` outside the active loop scope

### Loop Errors

- `repeat_until(max_iters < 1)`
- unsupported predicate type

## Validation Commands

## Current CLI Surface

The current CLI supports:

```sh
daiag run --workflow <path> [--param key=value] [--workdir <path>] [--verbose]
```

There is no dedicated `daiag validate` command today.

## Practical Validation Command

The current validation path is:

```sh
go run ./cmd/daiag run --workflow examples/poem/workflows/poem.star --param name=rain
```

or, after building:

```sh
go build ./cmd/daiag
./daiag run --workflow examples/poem/workflows/poem.star --param name=rain
```

Behavior:

- workflow loading happens first
- `load(...)` resolution happens first
- workflow validation happens before the first task starts
- if validation fails, execution exits with code `1`

Important limitation:

- `run` is not a validate-only command
- if validation succeeds, the workflow will execute tasks

So today:

- use `run` to validate and execute
- there is no side-effect-free validation subcommand yet

## Minimal Correct Example

```python
load("lib/paths.star", "feature_paths")
load("lib/tasks.star", "default_executor", "write_poem_task", "extend_poem_task", "review_poem_task")

name = param("name")
paths = feature_paths(name)

wf = workflow(
    id = "poem",
    default_executor = default_executor,
    steps = [
        write_poem_task(paths),
        repeat_until(
            id = "extend_until_ready",
            max_iters = 4,
            steps = [
                extend_poem_task(paths),
                review_poem_task(paths),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
```

## Authoring Recommendations

- keep `param(...)` near the entry file top level
- use `load(...)` for path helpers and task constructors
- keep prompt templates in separate Markdown files
- use `path_ref(...)` for file handoff between tasks
- use `json_ref(...)` only for control flow and metadata
- use `loop_iter(...)` only when you need per-iteration file names
- prefer helper functions over copying large task blocks

## Related Documents

- `docs/workflow-runner.md`
- `docs/starlark-load-design.md`
- `docs/loop-iteration-design.md`
