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
- Relative `--run-dir` values are resolved from `projectdir`.
- The CLI creates the directory before execution.
- For the first implementation, `run-dir` must be inside `projectdir`.
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
- Relative `--run-dir` and `--workflows-lib` paths resolve from `projectdir`,
  not from the process current working directory.
- Executors run from `projectdir`, not from the run artifact directory.

`--param` compatibility is outside this design. New workflows should continue
to use `workflow(inputs = [...])` and `--input`.

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
  --description "<workflow request>" \
  [--workflow <bootstrap-workflow-id>] \
  [--projectdir <path>] \
  [--run-dir <path>] \
  [--workflows-lib <dir>] \
  [--verbose]
```

Optional future convenience:

```sh
daiag bootstrap --description-file docs/new-workflow.md
```

`--description` is enough for the first implementation.

## Bootstrap Behavior

`daiag bootstrap` is a convenience command over normal workflow execution.

Recommended first implementation:

1. Resolve `projectdir`, `run-dir`, and `workflows-lib`.
2. Run the selected catalog bootstrap workflow. Default to
   `workflow_bootstrapper`.
3. Pass the user request as `--input description=<description>`.
4. Execute tasks with executor CWD set to `projectdir`.
5. Resolve relative declared artifacts against `run-dir`.
6. Let the workflow-authoring task write generated workflow files directly under
   `workflows-lib` as project workspace edits.
7. Read the bootstrap workflow output results from the runtime result.
8. Require `outcome = "complete"`.
9. Validate the generated workflow ID from `workflows-lib`.
10. Print the generated workflow ID, workflow path, and run directory.

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

Output artifacts:

- `blueprint` - the planning artifact under `run-dir`
- `summary` - the authoring summary under `run-dir`

Output results:

- `workflow_id` - generated workflow ID
- `workflow_path` - generated workflow `.star` path
- `outcome` - `complete` or `needs_clarification`

The concrete bootstrap workflow implementation is intentionally outside this
CLI spec. A bootstrap workflow may compose any authoring workflow or task
sequence as long as it satisfies this input, artifact, and result contract.

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

- Relative task artifacts resolve against `RunDir`.
- Relative workflow output artifacts resolve against `RunDir`.
- Absolute artifact paths are preserved.
- For a first implementation, absolute artifact paths should be under
  `ProjectDir` or `RunDir`; paths outside those roots should fail validation or
  execution with a clear error.

Runtime result behavior:

- After all top-level steps complete, resolve the workflow's
  `output_artifacts` and `output_results`.
- Return them to the CLI in a `RunResult`.
- `daiag run` may print a concise summary.
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
project_catalog = format(
    "{project}/.daiag/workflows/WORKFLOWS.md",
    project = projectdir(),
)
summary_path = format(
    "{run_dir}/workflow_author_from_blueprint/summary.md",
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

In most bootstrap prompts, relative repo paths such as
`.daiag/workflows/WORKFLOWS.md` are also valid because the executor CWD is
`projectdir`.

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
run-dir:       /Users/nik/Projects/daiag/.daiag/runs/bootstrap/20260413-123000
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
.daiag/runs/bootstrap/20260413-123000/workflow_composer/blueprint.md
.daiag/runs/bootstrap/20260413-123000/workflow_author_from_blueprint/summary.md
```

## Implementation Plan

1. Add CLI path resolution for `--projectdir`, `--run-dir`, and
   `--workflows-lib`.
2. Add `--workflow` support to `bootstrap`, defaulting to
   `workflow_bootstrapper`.
3. Remove `--workdir` from `run` usage and parsing.
4. Add `--input` support to `validate`.
5. Change runtime inputs and executor requests from `Workdir` to `ProjectDir`
   and `RunDir`.
6. Resolve relative artifacts against `RunDir`.
7. Run executors from `ProjectDir`.
8. Add `RunResult` with top-level workflow outputs.
9. Change `projectdir()` to use the CLI project directory.
10. Replace `workdir()` with `run_dir()` in the DSL and docs.
11. Add or update the default bootstrap workflow.
12. Ensure bootstrap workflows return `workflow_id` so validation does not have
    to infer the ID from a path.
13. Update bootstrap authoring workflows so prompt write paths use
    `run_dir()`-rooted values or `path_ref(...)` values.
14. Add `.daiag/runs/` to `.gitignore`.
15. Add tests for path defaults, relative path resolution, artifact resolution,
    executor CWD, bootstrap validation, and validate inputs.

## Open Questions

- Should `run-dir` outside `projectdir` be supported later by adding executor
  writable roots?
- Should `daiag bootstrap` accept `--description-file` in the first version or
  wait until the simple `--description` flow works?
- Should `daiag run` print top-level workflow outputs by default, or only with a
  machine-readable output flag?
