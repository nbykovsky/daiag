# CLI Reference

## Purpose

This document is the reference for the `daiag` command-line interface.

## Synopsis

```sh
daiag <command> [flags]
```

Commands:

- `run` — execute a workflow
- `help` — print usage

## `daiag run`

Loads and executes a workflow.

```sh
daiag run --workflow <workflow-id> --workdir <path> [--workflows-lib <dir>] [--input key=value]... [--param key=value]... [--verbose]
```

### Flags

#### `--workflow <workflow-id>` (required)

The workflow ID to run. Must match `[A-Za-z0-9_-]+`. Resolves to
`<workflows-lib>/<id>/<id>.star` at load time.

Path-style values such as `./wf.star`, `../wf.star`, or `/abs/wf.star` are
rejected.

#### `--workdir <path>` (required)

Absolute path to the run output directory.

- Must be an absolute path.
- Created with `mkdir -p` before execution begins if it does not exist.
- Shared across all subworkflows in the run.
- Used as the executor CWD for every task.

#### `--workflows-lib <dir>` (optional)

Path to the workflows library directory.

- Relative paths are resolved from the process current working directory.
- When omitted, defaults to `<projectdir>/.daiag/workflows` where
  `<projectdir>` is found by walking up from the process current working
  directory until a `.daiag/` directory is found.
- Fails at startup if supplied and the path does not exist or is not a
  directory.
- Fails if omitted and no `.daiag/` ancestor can be found.

#### `--input key=value` (repeatable)

Supplies a workflow input for `input(...)` builtins declared in
`workflow(inputs = [...])`.

May be repeated to supply multiple inputs:

```sh
daiag run --workflow feature-development --workdir /tmp/run \
  --input name=indicators \
  --input env=staging
```

#### `--param key=value` (repeatable)

Supplies a compatibility parameter for `param(...)` builtins in existing
top-level workflows. May be repeated.

`--input` and `--param` values are merged into a single map. If the same key
appears in both flags with different values, the run fails with an error.

New workflows should use `workflow(inputs = [...])` and `--input` instead.

#### `--verbose`

Enables verbose output. Off by default.

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Workflow load or execution error |
| 2 | Argument error or unknown command |

### Examples

Run a workflow from the default library:

```sh
daiag run --workflow write_poem --workdir /output/run1
```

Run a workflow from an explicit library:

```sh
daiag run --workflow feature-development \
  --workflows-lib examples/development-workflow/workflows \
  --workdir /tmp/daiag-run \
  --input name=indicators
```

Run a workflow from a shared or external library:

```sh
daiag run --workflow spec-refinement \
  --workflows-lib /shared/workflows \
  --workdir /output/run1 \
  --input feature_dir=/projects/myapp/docs/features/login
```

Run from an experiment library with a relative path:

```sh
daiag run --workflow draft --workflows-lib ./experiments --workdir /output
```

## `daiag help`

Prints usage to stdout and exits with code 0. Also triggered by `-h` or
`--help` as the first argument.

```sh
daiag help
daiag -h
daiag --help
```

## Error Output

Errors are written to stderr. The exit code indicates the error class:

- Argument errors (missing required flags, invalid values, unknown commands)
  print the error followed by the usage summary and exit with code 2.
- Workflow load and execution errors print the error and exit with code 1
  without reprinting usage.
