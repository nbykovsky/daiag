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

- Add `--workflows-lib` CLI flag with a default of `.daiag/workflows/` when
  name resolution is used.
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
daiag run --workflow <name-or-path> --workdir <absolute-path> [--workflows-lib <dir>]
```

`--workdir` remains required and must be an absolute path.

When `--workflows-lib` is omitted and name resolution is needed, the default
library is `.daiag/workflows/` under the project root. The project root is found
by walking up from the process current working directory until a directory
containing `.daiag/` is found.

If no `.daiag/` ancestor is found, named workflow resolution fails unless
`--workflows-lib` is supplied.

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

A `--workflow` value or `subworkflow(workflow = ...)` value is treated as a
**name** only if it matches:

```text
[A-Za-z0-9_-]+
```

Otherwise it is treated as a **path**.

Names must not be empty, `.` or `..`, contain path separators, or end in
`.star`.

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

When a name is used, the loader resolves it against the workflows library.
The workflow author does not need to know or encode the library location.

### Module Boundaries

Module boundaries depend on how the workflow was resolved:

| Workflow reference | Allowed module boundary |
|---|---|
| entry workflow by name | workflows library root |
| entry workflow by path | entry workflow directory |
| subworkflow by name | workflows library root |
| subworkflow by path | caller workflow's current boundary |

Relative `load(...)` and path-style `subworkflow(...)` references are resolved
from the calling `.star` module and must remain inside that workflow's allowed
boundary.

### Default Library Location

The default library is resolved relative to the project root:

```
<projectdir>/.daiag/workflows/
```

`<projectdir>` is found by walking up from the process current working directory
until a directory containing `.daiag/` is found.

If `--workflows-lib` is supplied, it overrides this default. The flag accepts
both absolute and relative paths; relative paths are resolved from the process
current working directory.

## Validation Rules

- If `--workflow` is a name and the workflows library does not contain the
  expected `<name>/<name>.star` file, fail with a clear error listing the
  expected path.
- If `subworkflow(workflow = ...)` is a name and the workflows library does
  not contain the expected `<name>/<name>.star` file, fail with a clear error
  listing the expected path.
- If `--workflows-lib` is explicitly supplied and does not exist, fail at
  startup with a clear error.
- If `--workflows-lib` is omitted and name resolution is needed, but no
  `.daiag/` ancestor can be found from the process current working directory,
  fail with a clear error.
- Path-style values bypass workflow name resolution and follow existing load
  path rules.

## Implementation Tasks

1. Add `--workflows-lib` flag to the CLI in `internal/cli`; when omitted and
   name resolution is needed, default to `<projectdir>/.daiag/workflows/` where
   `<projectdir>` is found by walking up from the process current working
   directory.
2. Add name resolution logic: if the value matches `[A-Za-z0-9_-]+`, expand to
   `<workflows-lib>/<name>/<name>.star`.
3. Apply the same resolution when evaluating `subworkflow(workflow = ...)` in
   `internal/starlarkdsl`.
4. Track an allowed module boundary per loaded workflow: workflows library root
   for name references, entry workflow directory or caller boundary for path
   references.
5. Add validation: explicitly supplied missing library directory, missing
   default project root during name resolution, and unresolvable names are load
   time errors.
6. Update `docs/workflow-language.md` to document name resolution and the
   `--workflows-lib` flag.
