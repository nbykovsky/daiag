# CLI Reference

## Purpose

This document describes the implemented `daiag` command-line interface.

## Synopsis

```sh
daiag <command> [flags]
```

Commands:

- `run` — execute a workflow
- `validate` — parse and validate a workflow without executing tasks
- `bootstrap` — generate a workflow through a catalog bootstrap workflow
- `help` — print usage

## Path Model

`daiag` separates three paths:

- `projectdir` — editable project workspace and executor current directory
- `run-dir` — root for declared run artifacts
- `workflows-lib` — Starlark workflow library root

Defaults:

- `--projectdir` defaults to the nearest ancestor of the process current working directory that contains `.daiag/`.
- `--workflows-lib` defaults to `<projectdir>/.daiag/workflows`.
- `daiag run` defaults `--run-dir` to `.daiag/runs/<workflow-id>/<timestamp>` under `projectdir`.
- `daiag bootstrap` defaults `--run-dir` to `.daiag/runs/bootstrap/<timestamp>` under `projectdir`.

Default run directory timestamps use UTC in `YYYYMMDD-HHMMSS-NNNNNNNNNZ`
format. If a generated directory already exists, the CLI appends a numeric
suffix before retrying.

Relative `--run-dir` and `--workflows-lib` values resolve from `projectdir`.
Relative `--projectdir` and `--description-file` values resolve from the
process current working directory.

The CLI creates `run-dir` before execution. `run-dir` must be inside
`projectdir`. `workflows-lib` must exist. `bootstrap` also requires
`workflows-lib` to be inside `projectdir`.

## `daiag run`

Loads and executes a workflow.

```sh
daiag run \
  --workflow <workflow-id> \
  [--projectdir <path>] \
  [--run-dir <path>] \
  [--workflows-lib <dir>] \
  [--input key=value]... \
  [--verbose]
```

`--workflow` is required and must match `[A-Za-z0-9_-]+`. It resolves to
`<workflows-lib>/<workflow-id>/workflow.star`.

`--input key=value` may be repeated. `--param` and `--workdir` are not
supported CLI flags.

After a successful run, top-level workflow outputs are printed when present:

```text
workflow outputs:
artifact <key>: <absolute-path>
result <key>: <json>
```

Artifact keys and result keys are sorted independently.

Example:

```sh
daiag run --workflow poem_generator --input n=6
```

Example with explicit paths:

```sh
daiag run \
  --workflow feature-development \
  --projectdir /projects/myapp \
  --workflows-lib examples/development-workflow/workflows \
  --run-dir .daiag/runs/feature-development/manual \
  --input name=indicators
```

## `daiag validate`

Loads and validates a workflow without executing tasks.

```sh
daiag validate \
  --workflow <workflow-id> \
  [--projectdir <path>] \
  [--workflows-lib <dir>] \
  [--input key=value]...
```

`--input` is accepted for workflows or legacy top-level `param(...)` references
that need load-time values. Declared `input(...)` values do not need concrete
values for validation.

Examples:

```sh
daiag validate --workflow workflow_bootstrapper
```

```sh
daiag validate \
  --workflow feature-development \
  --projectdir /projects/myapp \
  --workflows-lib examples/development-workflow/workflows \
  --input name=indicators
```

## `daiag bootstrap`

Runs a bootstrap workflow from the selected catalog and validates the generated
workflow.

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
`--workflow` selects the bootstrap workflow to execute and defaults to
`workflow_bootstrapper`.

The bootstrap command passes these inputs to the selected workflow:

- `description`
- `workflows_lib`

The selected workflow must return string output results:

- `workflow_id`
- `workflow_path`
- `outcome`

`outcome` must be `complete`. `workflow_path` must be absolute and must match
`<workflows-lib>/<workflow_id>/workflow.star` after symlink resolution. The CLI
then validates the generated workflow from the selected `workflows-lib`.

Success output:

```text
bootstrap complete
workflow: <workflow-id>
workflow path: <absolute-workflow-path>
run dir: <absolute-run-dir>
```

Example:

```sh
daiag bootstrap --description "create a workflow that writes a haiku and reviews it"
```

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Workflow load, validation, or execution error |
| 2 | Argument error or unknown command |

Errors are written to stderr. Argument errors also print the usage summary.

## Help

```sh
daiag help
daiag -h
daiag --help
```
