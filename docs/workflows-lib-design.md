# Workflows Library Design

## Purpose

This document specifies the `--workflows-lib` flag and workflow ID resolution
for `daiag`.

A workflows library is a directory that holds workflow subdirectories following
the `<workflow_id>/<workflow_id>.star` convention. Naming a workflow by its ID
rather than its file path makes CLI commands and `subworkflow(...)` references
shorter and location-independent. Workflow file paths are not accepted as
workflow references in this design.

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
- Require `--workflow` to be a workflow ID.
- Require `subworkflow(workflow = ...)` to be a workflow ID.
- Resolve workflow IDs to `<workflows-lib>/<id>/<id>.star` at load time.
- Keep workflow module loading under one workflows library boundary.

## Non-Goals

- Multiple library paths or search order across several directories.
- Automatic discovery or listing of workflows (separate concern).
- Running ad hoc workflow files directly by path. Use `--workflows-lib` to
  point at an alternate library root instead.
- Remapping `projectdir()` for shared workflow libraries outside the calling
  project tree.

## Design

### CLI Flags

```sh
daiag run --workflow <workflow-id> --workdir <absolute-path> [--workflows-lib <dir>]
```

`--workdir` remains required and must be an absolute path.

When `--workflows-lib` is omitted, the default library is `.daiag/workflows/`
under the project root. The project root is found by walking up from the
process current working directory until a directory
containing `.daiag/` is found.

If no `.daiag/` ancestor is found, workflow ID resolution fails unless
`--workflows-lib` is supplied.

Examples:

```sh
# Resolve write_poem from the default library
daiag run --workflow write_poem --workdir /output/run1

# Resolve from a custom library location
daiag run --workflow write_poem --workflows-lib /shared/workflows --workdir /output

# Resolve draft from an experiment library
daiag run --workflow draft --workflows-lib ./experiments --workdir /output
```

### Workflow ID Resolution

A `--workflow` value or `subworkflow(workflow = ...)` value must be a workflow
ID matching:

```text
[A-Za-z0-9_-]+
```

IDs must not be empty, `.` or `..`, contain path separators, or end in
`.star`.

| Value | Treatment | Resolved to |
|---|---|---|
| `write_poem` | workflow ID | `<workflows-lib>/write_poem/write_poem.star` |
| `draft` with `--workflows-lib ./experiments` | workflow ID | `./experiments/draft/draft.star` |
| `./path/to/wf.star` | invalid | error |
| `../sibling/wf.star` | invalid | error |
| `/absolute/wf.star` | invalid | error |

### `subworkflow(workflow = ...)` Resolution

The same workflow ID rule applies inside the DSL:

```python
subworkflow(
    id = "spec_refinement",
    workflow = "spec_refinement",
    inputs = {...},
)
```

The loader resolves the workflow ID against the workflows library.
The workflow author does not need to know or encode the library location.

### Module Boundaries

All entry workflows and subworkflows are resolved from the workflows library.
The workflows library root is the allowed module boundary.

Relative `load(...)` references are resolved from the calling `.star` module and
must remain inside the workflows library root. `subworkflow(...)` references are
workflow IDs, not module paths, so they are always resolved from the workflows
library root.

This replaces the current single "workflow base directory" concept for module
validation. Existing docs and code currently use one workflow base directory
equal to the entry workflow file's directory. After this change, validation
should talk about the workflows library root as the allowed module boundary.

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

### `projectdir()` in External Libraries

`projectdir()` continues to resolve from the `.star` module where it is called.
It does not resolve relative to the process current working directory or the
project that invoked the workflow.

If `--workflows-lib` points outside the project tree, for example
`/shared/workflows`, workflows in that library may not have a `.daiag/`
ancestor. In that case, `projectdir()` fails at load time using the existing
`projectdir()` error path. Shared workflows that need a caller project path
should accept it as an explicit workflow input.

## Validation Rules

These rules replace the current path-only workflow reference validation in
`docs/workflow-language.md`.

- `--workflow` must be a valid workflow ID.
- `subworkflow(workflow = ...)` must be a valid workflow ID.
- Workflow IDs resolve to `<workflows-lib>/<id>/<id>.star`; the resolved
  file must exist.
- If `--workflow` or `subworkflow(workflow = ...)` contains a path separator,
  ends in `.star`, contains `://`, or otherwise fails the ID pattern, fail with
  a clear error explaining that workflow references must be IDs.
- If `--workflow` is an ID and the workflows library does not contain the
  expected `<id>/<id>.star` file, fail with a clear error listing the expected
  path.
- If `subworkflow(workflow = ...)` is an ID and the workflows library does not
  contain the expected `<id>/<id>.star` file, fail with a clear error listing
  the expected path.
- If `--workflows-lib` is explicitly supplied and does not exist, fail at
  startup with a clear error.
- If `--workflows-lib` is omitted, but no `.daiag/` ancestor can be found from
  the process current working directory, fail with a clear error.

## Implementation Tasks

1. Add `--workflows-lib` flag to the CLI in `internal/cli`; when omitted and
   no explicit library is supplied, default to `<projectdir>/.daiag/workflows/`
   where `<projectdir>` is found by walking up from the process current working
   directory.
2. Add workflow ID validation and resolution: values must match
   `[A-Za-z0-9_-]+` and expand to `<workflows-lib>/<id>/<id>.star`.
3. Apply the same resolution when evaluating `subworkflow(workflow = ...)` in
   `internal/starlarkdsl`.
4. Use the workflows library root as the loader and module-validation boundary
   for entry workflows, subworkflows, and relative `load(...)` references.
5. Add validation: explicitly supplied missing library directory, missing
   default project root during workflow ID resolution, invalid workflow IDs, and
   unresolvable IDs are load time errors.
6. Update `docs/workflow-language.md` to document workflow ID resolution and the
   `--workflows-lib` flag, including:
   - Current CLI Surface
   - Module Loading rules
   - `subworkflow(...)` rules
   - Subworkflow Errors
   - any remaining "workflow base directory" wording

## Test Expectations

Add focused tests for:

- CLI parsing of `--workflows-lib`
- workflow ID validation
- default workflows library discovery from process current working directory
- explicit `--workflows-lib` relative and absolute paths
- entry workflow resolution to `<workflows-lib>/<id>/<id>.star`
- subworkflow resolution to `<workflows-lib>/<id>/<id>.star`
- path-style entry and subworkflow references rejected
- missing default project root when workflow ID resolution needs the default
  workflows library
- missing explicitly supplied workflows library
- missing workflow ID file with an error listing the expected path
- boundary enforcement for relative `load(...)` references under the workflows
  library root
- `projectdir()` failure in an external workflows library with no `.daiag/`
  ancestor
