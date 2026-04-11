# Workflows Library Design

## Purpose

This document specifies the `--workflows-lib` flag and workflow name resolution
for `daiag`.

A workflows library is a directory that holds workflow subdirectories following
the `<workflow_id>/<workflow_id>.star` convention. Naming a workflow by its ID
rather than its file path makes CLI commands and `subworkflow(...)` references
shorter and location-independent.

This is an implementation design, not current behavior.

## Problem

Today workflows are referenced by file path everywhere:

```sh
daiag run --workflow .daiag/workflows/write_poem/write_poem.star
```

```python
subworkflow(
    id = "spec_refinement",
    workflow = "../spec_refinement/spec_refinement.star",
    ...
)
```

This hardcodes the library location in every CLI invocation and in every
`subworkflow(...)` call. Moving or reorganising the library breaks all
references.

## Goals

- Add `--workflows-lib` CLI flag with a default of `.daiag/workflows/`.
- Allow `--workflow` to accept either a workflow ID (name) or a file path.
- Allow `subworkflow(workflow = ...)` to accept either a workflow ID or a path.
- Resolve workflow IDs to `<workflows-lib>/<id>/<id>.star` at load time.
- Keep full path references working as an escape hatch.

## Non-Goals

- Multiple library paths or search order across several directories.
- Automatic discovery or listing of workflows (separate concern).

## Design

### CLI Flags

```sh
daiag run --workflow <name-or-path> [--workflows-lib <dir>] [--workdir <dir>]
```

`--workflows-lib` defaults to `.daiag/workflows/` relative to the project root
(the directory containing `.daiag/`).

Examples:

```sh
# Resolve write_poem from the default library
daiag run --workflow write_poem --workdir /output/run1

# Resolve from a custom library location
daiag run --workflow write_poem --workflows-lib /shared/workflows --workdir /output

# Use a full path directly — bypasses the library
daiag run --workflow ./experiments/draft.star --workdir /output
```

### Workflow Name Resolution

A `--workflow` value (or `subworkflow(workflow = ...)` value) is treated as a
**name** if it contains no `/` and does not end in `.star`. Otherwise it is
treated as a **path**.

| Value | Treatment | Resolved to |
|---|---|---|
| `write_poem` | name | `<workflows-lib>/write_poem/write_poem.star` |
| `./path/to/wf.star` | path | used as-is |
| `../sibling/wf.star` | path | resolved relative to caller |
| `/absolute/wf.star` | path | used as-is |

### `subworkflow(workflow = ...)` Resolution

The same name-vs-path rule applies inside the DSL:

```python
# Name — resolved from the workflows library
subworkflow(
    id = "spec_refinement",
    workflow = "spec_refinement",
    inputs = {...},
)

# Path — resolved relative to the calling .star file (existing behavior)
subworkflow(
    id = "spec_refinement",
    workflow = "../spec_refinement/spec_refinement.star",
    inputs = {...},
)
```

When a name is used, the library path is injected by the runtime at load time.
The workflow author does not need to know or encode the library location.

### Default Library Location

The default library is resolved relative to the project root — the directory
containing `.daiag/`:

```
<projectdir>/.daiag/workflows/
```

If `--workflows-lib` is supplied it overrides this default. The flag accepts
both absolute and relative paths; relative paths are resolved from the current
working directory.

## Validation Rules

- If `--workflow` is a name and `--workflows-lib` does not contain a matching
  subdirectory, fail with a clear error listing the expected path.
- If `--workflows-lib` does not exist, fail at startup with a clear error.
- Path-style values bypass the library and follow existing load path rules.

## Implementation Tasks

1. Add `--workflows-lib` flag to the CLI in `internal/cli`; default to
   `<projectdir>/.daiag/workflows/`.
2. Add name resolution logic: if the value has no `/` and no `.star` suffix,
   expand to `<workflows-lib>/<name>/<name>.star`.
3. Apply the same resolution when evaluating `subworkflow(workflow = ...)` in
   `internal/starlarkdsl`.
4. Add validation: missing library directory and unresolvable names are load
   time errors.
5. Update `docs/workflow-language.md` to document name resolution and the
   `--workflows-lib` flag.
