# Workflow Bootstrap CLI Design

## Purpose

This document proposes a breaking CLI path model that makes workflow
bootstrapping convenient.

The desired user flow is:

```sh
daiag bootstrap --description "create a workflow that ..."
```

When the command succeeds, the generated workflow source files should appear in
the selected workflow catalog. By default this is the project workflow catalog:

```text
.daiag/workflows/<workflow-id>/workflow.star
.daiag/workflows/<workflow-id>/<task-id>.md
.daiag/workflows/WORKFLOWS.md
```

Run scratch files such as blueprints and summaries should not be mixed into the
repository root or the workflow catalog. They should be written under a run
directory such as:

```text
.daiag/runs/bootstrap/<run-id>/
```

## Problem

The current `--workdir` flag has too many meanings:

- run artifact root
- executor current working directory
- executor writable workspace
- path returned by `workdir()`

That is awkward for bootstrapping.

If `--workdir` is set to `.daiag/workflows`, relative run artifacts land near
the workflow catalog, but the executor starts inside `.daiag/workflows`. Prompts
that refer to repo-root paths such as `.daiag/workflows/WORKFLOWS.md` then point
at the wrong nested location.

If `--workdir` is set to the project root, repo-root paths work, but normal run
artifacts such as `workflow_composer/blueprint.md` are written into the root of
the working tree.

Bootstrapping needs these concepts to be separate:

- the project workspace that the agent can edit
- the run directory where declared workflow artifacts go
- the workflow library where `.star` files are loaded from

## Goals

- Make `daiag bootstrap --description "..."` the convenient default.
- Run executors from the project root so `.daiag/...` paths mean repository
  paths.
- Store ordinary run artifacts under `.daiag/runs/...`.
- Load workflow definitions from `.daiag/workflows` by default.
- Allow workflows to create or update files in the selected workflow catalog as
  normal workspace edits.
- Validate the newly generated workflow after the bootstrap workflow completes.
- Keep resolver behavior simple: one run uses one workflow library.
- Do not preserve compatibility with the old `--workdir` behavior.

## Non-Goals

- Do not stage a full workflow library copy and apply it later.
- Do not add a workflow-language primitive for creating workflows.
- Do not make `daiag run` dynamically reload workflows during a single already
  loaded workflow graph.
- Do not solve undo or review flows in the first implementation. Use Git for
  reviewing or reverting direct workspace edits.
- Do not support `run-dir` outside `projectdir`; keeping run artifacts inside
  the project avoids extra executor writable roots.

## New Path Model

Introduce three explicit paths.

### Project Directory

`projectdir` is the editable project workspace.

Rules:

- CLI flag: `--projectdir <path>`
- Default: nearest ancestor of the process current working directory that
  contains `.daiag/`
- Relative `--projectdir` values are resolved from the process current working
  directory.
- The directory must exist.
- Executors run with this directory as their current working directory.
- Executor writable access is rooted here for the first implementation.
- `projectdir()` resolves to this path.

### Run Directory

`run-dir` is the root for declared workflow artifacts.

Rules:

- CLI flag: `--run-dir <path>`
- Default for `daiag run`:
  `.daiag/runs/<workflow-id>/<timestamp>`
- Default for `daiag bootstrap`:
  `.daiag/runs/bootstrap/<timestamp>`
- The default timestamp uses UTC with nanosecond precision:
  `YYYYMMDD-HHMMSS-NNNNNNNNNZ`.
- If the default run directory already exists, the CLI should retry with a
  fresh timestamp and then append an incrementing `-NN` suffix until it can
  create a new directory.
- Relative `--run-dir` values are resolved from `projectdir`.
- The CLI creates the directory before execution. An explicit `--run-dir` may
  already exist; the CLI should reuse it after validating that it is a
  directory inside `projectdir`.
- For the first implementation, `run-dir` must be inside `projectdir`.
- Containment checks use cleaned absolute paths after resolving symlinks.
- Relative `artifact(...)` paths and `output_artifacts` paths resolve against
  `run-dir`.
- A new `run_dir()` DSL builtin resolves to this path.

The repository should ignore `.daiag/runs/`.

### Workflow Library

`workflows-lib` is the Starlark workflow library root.

Rules:

- CLI flag: `--workflows-lib <path>`
- Default: `<projectdir>/.daiag/workflows`
- Relative `--workflows-lib` values are resolved from `projectdir`.
- The path must exist and must be a directory.
- Entry workflows still resolve as:

```text
<workflows-lib>/<workflow-id>/workflow.star
```

- `load(...)` and `subworkflow(...)` resolution remain scoped to this library.
- For `daiag bootstrap`, this is also the target catalog for generated
  workflow files and the library used for post-generation validation.
- Because the first implementation roots executor writes at `projectdir`,
  bootstrap should require `workflows-lib` to be inside `projectdir`.
- Containment checks use cleaned absolute paths after resolving symlinks.

## Command Shape

### `daiag run`

```sh
daiag run \
  --workflow <workflow-id> \
  [--projectdir <path>] \
  [--run-dir <path>] \
  [--workflows-lib <dir>] \
  [--input key=value]... \
  [--verbose]
```

Breaking changes:

- Remove `--workdir`.
- Remove `--param`; use `--input` for all workflow inputs.
- Relative `--run-dir` and `--workflows-lib` paths resolve from `projectdir`,
  not from the process current working directory.
- Executors run from `projectdir`, not from the run artifact directory.

### `daiag validate`

```sh
daiag validate \
  --workflow <workflow-id> \
  [--projectdir <path>] \
  [--workflows-lib <dir>] \
  [--input key=value]...
```

Validation should accept `--input` so workflows with declared inputs can be
validated without running tasks.

Validation does not need `--run-dir` because it does not execute tasks or check
runtime artifact existence.

### `daiag bootstrap`

```sh
daiag bootstrap \
  (--description "<workflow request>" | --description-file <path>) \
  [--workflow <bootstrap-workflow-id>] \
  [--projectdir <path>] \
  [--run-dir <path>] \
  [--workflows-lib <dir>] \
  [--verbose]
```

Exactly one of `--description` or `--description-file` is required.
`--description-file` reads the workflow request from a UTF-8 text file.
Relative `--description-file` values are resolved from the process current
working directory.

## Bootstrap Behavior

`daiag bootstrap` is a convenience command over normal workflow execution.

Recommended first implementation:

1. Resolve `projectdir`, `run-dir`, and `workflows-lib`.
2. Run the selected catalog bootstrap workflow. Default to
   `workflow_bootstrapper`.
3. Read the workflow request from `--description` or `--description-file`.
4. Pass `description=<description>` and `workflows_lib=<abs workflows-lib>` as
   bootstrap workflow inputs.
5. Execute tasks with executor CWD set to `projectdir`.
6. Resolve relative declared artifacts against `run-dir`.
7. Let the workflow-authoring task write generated workflow files directly under
   `workflows-lib` as project workspace edits.
8. Read the bootstrap workflow output results from the runtime result.
9. Require `outcome = "complete"`.
10. Validate the generated workflow ID from `workflows-lib`.
11. Print the generated workflow ID, workflow path, and run directory.

Expected output shape:

```text
bootstrap complete
workflow: <workflow-id>
workflow path: <workflows-lib>/<workflow-id>/workflow.star
run dir: /abs/project/.daiag/runs/bootstrap/<run-id>
```

If validation fails, the command should exit non-zero and leave the workspace
unchanged beyond whatever files the executor already wrote. The user can inspect
or revert those files with Git.

## Bootstrap Workflow Contract

The bootstrap command should invoke a normal workflow so most orchestration
stays in the workflow catalog.

The default bootstrap workflow ID should be:

```text
workflow_bootstrapper
```

The `--workflow <bootstrap-workflow-id>` flag selects which bootstrap workflow
to execute. It does not name the workflow being generated. This keeps
`daiag bootstrap --description "..."` convenient while allowing alternate
bootstrap pipelines such as faster, more reviewed, or issue-driven variants.

Every bootstrap workflow selected by this flag should follow the same contract.

Input:

- `description` - the user's natural-language workflow request
- `workflows_lib` - absolute path to the selected workflow catalog where the
  bootstrap workflow should write generated workflow files

Output artifacts:

- `blueprint` - the planning artifact under `run-dir`
- `summary` - the authoring summary under `run-dir`

Output results:

- `workflow_id` - generated workflow ID
- `workflow_path` - generated workflow `.star` path
- `outcome` - `complete` or `needs_clarification`

`workflow_id`, `workflow_path`, and `outcome` are required string results. The
CLI should fail if any are missing or are not strings. It should not infer
`workflow_id` from `workflow_path`.

`workflow_path` must be absolute. After cleaning both paths and resolving
symlinks, it must match `<resolved workflows-lib>/<workflow_id>/workflow.star`.
If it does not match, the CLI should fail before validation.

The concrete bootstrap workflow implementation is intentionally outside this
CLI spec. A bootstrap workflow may compose any authoring workflow or task
sequence as long as it satisfies this input, artifact, and result contract.
Any existing authoring workflow used by the default bootstrap workflow must be
updated or wrapped to return `workflow_id`.

## Runtime Changes

The runtime should carry both project and run paths.

Replace:

```go
type RunInput struct {
    Workdir string
}

type TaskRequest struct {
    Workdir string
}
```

With:

```go
type RunInput struct {
    ProjectDir string
    RunDir     string
}

type TaskRequest struct {
    ProjectDir string
    RunDir     string
}
```

Executor behavior:

- Codex uses `ProjectDir` for `-C` and process `Dir`.
- Claude uses `ProjectDir` for process `Dir` and writable project access.
- `RunDir` is available only for prompt rendering and artifact resolution.
- Because the first implementation requires `RunDir` under `ProjectDir`, no
  extra executor writable root is needed.

Artifact behavior:

- Relative task artifacts and relative workflow output artifacts resolve by
  joining the cleaned relative path to `RunDir`.
- After clean/join and symlink resolution, relative artifact paths must remain
  under `RunDir`. A path such as `artifact("../foo.md")` is invalid even if the
  resolved file would still be under `ProjectDir`.
- Absolute artifact paths are preserved, but after cleaning and symlink
  resolution they must remain under `ProjectDir`.
- Paths outside the allowed root should fail validation or execution with a
  clear error.
- Static validation should check literal absolute artifact paths when the path
  is known without runtime data.
- Runtime validation should check every resolved artifact path after evaluating
  `input(...)`, `format(...)`, `path_ref(...)`, `json_ref(...)`, `run_dir()`,
  or `projectdir()`. This catches computed paths and relative paths containing
  `..`.

Runtime result behavior:

- After all top-level steps complete, resolve the workflow's
  `output_artifacts` and `output_results`.
- Return them to the CLI in a `RunResult`:

```go
type RunResult struct {
    EntryWorkflowID   string
    EntryWorkflowPath string
    ProjectDir       string
    RunDir           string
    OutputArtifacts  map[string]string
    OutputResults    map[string]any
}
```

- `OutputArtifacts` contains resolved absolute artifact paths.
- `OutputResults` contains JSON-compatible values from `output_results`.
- If any step fails, return a non-nil error and do not return a partial
  successful `RunResult`.
- If any top-level output cannot be resolved, return a non-nil error with the
  output key in the error context and do not return a partial successful
  `RunResult`.
- `daiag run` should print resolved top-level workflow outputs to stdout after
  a successful run, after normal progress messages. If there are no top-level
  outputs, it should not print an output summary.
- The human-readable output summary should start with `workflow outputs:`. Then
  print artifact lines first and result lines second.
- Artifact keys and result keys should each be sorted lexicographically.
- Artifact lines should use `artifact <key>: <absolute-path>`.
- Result lines should use `result <key>: <json>`, where `<json>` is the
  single-line JSON encoding of the resolved value.
- A future machine-readable flag such as `--format json` can reuse `RunResult`.
- `daiag bootstrap` must use the result to find `workflow_id`,
  `workflow_path`, and `outcome`.

## Workflow Language Changes

Replace `workdir()` with `run_dir()`.

Rules:

- `run_dir()` resolves to the CLI run directory at execution time.
- Relative `artifact(...)` paths are still valid for run outputs.
- Prompt variables are not implicitly resolved against `run-dir`. If a prompt
  tells an executor where to write a run artifact, pass a `run_dir()`-rooted
  absolute path or a `path_ref(...)` from an earlier step.
- Use `projectdir()` when a workflow needs an absolute path to project source.
- `projectdir()` resolves to the CLI project directory, not to the workflow
  module's nearest `.daiag/` ancestor.
- `template_file(...)` remains resolved relative to the Starlark module where
  the call appears.
- `load(...)` remains resolved under the workflow library root.

Example:

```python
workflows_lib = input("workflows_lib")
project_catalog = format(
    "{workflows_lib}/WORKFLOWS.md",
    workflows_lib = workflows_lib,
)
summary_path = format(
    "{run_dir}/author/summary.md",
    run_dir = run_dir(),
)

task(
    id = "author",
    prompt = template_file("author.md", vars = {
        "CATALOG_PATH": project_catalog,
        "SUMMARY_PATH": summary_path,
    }),
    artifacts = {"summary": artifact(summary_path)},
    result_keys = ["workflow_id", "workflow_path", "outcome"],
)
```

In bootstrap prompts that only target the default catalog, relative repo paths
such as `.daiag/workflows/WORKFLOWS.md` are also valid because the executor CWD
is `projectdir`. Prompts that need to honor `--workflows-lib` should use the
`workflows_lib` input.

## Example

Command:

```sh
cd /Users/nik/Projects/daiag
daiag bootstrap --description "create a workflow that writes a haiku and reviews it"
```

Execution paths:

```text
projectdir:    /Users/nik/Projects/daiag
workflows-lib: /Users/nik/Projects/daiag/.daiag/workflows
run-dir:       /Users/nik/Projects/daiag/.daiag/runs/bootstrap/20260413-123000-123456789Z
```

Expected project edits:

```text
.daiag/workflows/haiku_review/workflow.star
.daiag/workflows/haiku_review/write_haiku.md
.daiag/workflows/haiku_review/review_haiku.md
.daiag/workflows/WORKFLOWS.md
```

Expected run artifacts:

```text
.daiag/runs/bootstrap/20260413-123000-123456789Z/planner/blueprint.md
.daiag/runs/bootstrap/20260413-123000-123456789Z/author/summary.md
```

## Implementation Plan

1. Add CLI path resolution for `--projectdir`, `--run-dir`, and
   `--workflows-lib`.
2. Add `--workflow` support to `bootstrap`, defaulting to
   `workflow_bootstrapper`.
3. Add `--description-file` support to `bootstrap`.
4. Remove `--workdir` and `--param` from `run` usage and parsing.
5. Add `--input` support to `validate`.
6. Change runtime inputs and executor requests from `Workdir` to `ProjectDir`
   and `RunDir`.
7. Resolve relative artifacts against `RunDir`.
8. Run executors from `ProjectDir`.
9. Add `RunResult` with top-level workflow outputs.
10. Change `projectdir()` to use the CLI project directory.
11. Replace `workdir()` with `run_dir()` in the DSL and docs.
12. Add or update the default bootstrap workflow.
13. Ensure bootstrap workflows return `workflow_id` so validation does not have
    to infer the ID from a path.
14. Update bootstrap authoring workflows so prompt write paths use
    `run_dir()`-rooted values or `path_ref(...)` values.
15. Add `.daiag/runs/` to `.gitignore`.
16. Add tests for path defaults, relative path resolution, artifact resolution,
    executor CWD, bootstrap validation, and validate inputs.
