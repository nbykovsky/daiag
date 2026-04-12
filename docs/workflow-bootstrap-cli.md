# Workflow Bootstrap CLI Proposal

## Purpose

This document describes a proposed `daiag bootstrap` command for using existing
workflows to create and review new workflows.

The command is intentionally separate from `daiag run`. Normal workflow runs
should keep using one workflow library and one run workdir. Bootstrap adds a
small CLI-level coordinator that stages a complete workflow library, runs the
authoring pipeline against that staged copy, and optionally applies the result
back to the source workflow library.

## Command Shape

```sh
daiag bootstrap \
  --description "<casual workflow request>" \
  --workdir <path> \
  [--workflows-lib <dir>] \
  [--apply] \
  [--verbose]
```

Flags:

- `--description` - casual user request for the workflow to build
- `--workdir` - bootstrap run directory
- `--workflows-lib` - source workflow library; defaults to
  `<projectdir>/.daiag/workflows`
- `--apply` - copy the staged workflow library back to the source workflow
  library after review and validation pass
- `--verbose` - enable verbose progress output

Defaults:

- source workflow library: `<projectdir>/.daiag/workflows`
- staged workflow library: `<workdir>/workflows`
- `--apply`: false

## Staged Workflow Library

Bootstrap starts by copying the entire source workflow library into:

```text
<workdir>/workflows/
```

Example:

```text
<workdir>/workflows/
  WORKFLOWS.md
  poem_generator/
    workflow.star
    write_poem.md
  file_row_grower/
    workflow.star
    add_row.md
    count_rows.md
  generated_new_workflow/
    workflow.star
    task.md
```

The staged library is a complete workflow library. Bootstrap stages must use it
as their only workflow library:

```sh
daiag run --workflow <bootstrap-stage> \
  --workflows-lib <workdir>/workflows \
  --workdir <workdir>
```

Subworkflow resolution does not fall back to `.daiag/workflows`. If a staged
workflow references another workflow, that referenced workflow must exist under
`<workdir>/workflows`.

This keeps normal resolver behavior simple: one run uses exactly one workflow
library.

## Pipeline

Bootstrap runs the workflow-authoring pipeline against the staged library:

1. Run gap analysis and structured planning.
2. If missing primitive workflows exist, run the primitive workflow author to
   create them in `<workdir>/workflows`, then rerun gap analysis against the
   original request and updated staged catalog.
3. When no gaps remain, run the Starlark workflow assembler to create the
   composed workflow in `<workdir>/workflows`.
4. Run the composition reviewer against the original request, no-gap structured
   plan, generated `.star` file, and staged `WORKFLOWS.md` entry.
5. If the reviewer requests changes, return only to the Starlark workflow
   assembler. The assembler edits the composed workflow and staged
   `WORKFLOWS.md`, then the reviewer runs again.
6. When the reviewer approves, validate the generated workflow from
   `<workdir>/workflows`.

The bootstrap command should not apply staged files to the source workflow
library until all required stages, review, and validation pass.

## Apply Behavior

When `--apply` is not set:

- do not modify the source workflow library
- leave the staged library at `<workdir>/workflows`
- print the staged workflow library path
- print that no changes were applied

Example output shape:

```text
bootstrap complete
staged workflows: /tmp/daiag-bootstrap/workflows
not applied
```

When `--apply` is set:

- after review and validation pass, copy the whole staged workflow library back
  over the source workflow library
- print the source workflow library path
- print the generated or updated workflow ID

Example output shape:

```text
bootstrap complete
applied workflows: /Users/nik/Projects/daiag/.daiag/workflows
workflow: poem_translate_and_grow
```

The first implementation can copy the entire staged library back to the source
library. It does not need per-file manifests, partial apply behavior, or a
separate apply-only command. In the first version, `--apply` applies the staged
library at the end of the same successful bootstrap run.

## Safety Rules

- Refuse to apply if `<workdir>/workflows/WORKFLOWS.md` is missing.
- Refuse to apply if review does not approve the composed workflow.
- Refuse to apply if validation fails.
- Do not apply partial results if any bootstrap stage fails.
- Keep `daiag run` behavior unchanged.
- Keep resolver behavior unchanged: one workflow run uses exactly one
  `--workflows-lib`.

## Notes

Workflows that create workflows do not need a new language construct. They can
write workflow files like any other task writes artifacts. Bootstrap's special
behavior is operational: it stages a complete workflow library, reruns workflow
steps as the staged catalog changes, and optionally applies the final staged
library back to the source library.

Newly created workflows should be treated as available to later bootstrap runs
or later bootstrap stages after the CLI reloads the staged library. They should
not be expected to appear inside the already-loaded graph of a single normal
`daiag run` execution.
