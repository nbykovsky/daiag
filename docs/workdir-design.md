# Workdir Design

## Purpose

This document specifies the `workdir` feature for `daiag`.

`workdir` gives workflows a single root directory for all output artifacts.
It eliminates path boilerplate from workflow files and enforces that outputs
land in a predictable, user-controlled location.

This is an implementation design, not current behavior.

## Problem

Today artifact paths are plain strings declared inside workflows:

```python
poem_path = format("{workdir}/{name}/poem.md", workdir = param("workdir"), name = param("name"))

artifacts = {"poem": artifact(poem_path)}
```

This has several problems:

- Every workflow must declare `workdir` as an explicit input or param.
- Subworkflows must receive `workdir` as a passed input, adding noise to every composition.
- Nothing stops a workflow from writing outside the intended output directory.
- Paths computed inside prompt templates are not automatically rooted anywhere.

## Goals

- Provide `workdir()` as a zero-argument DSL builtin that returns the workdir at runtime.
- Resolve relative artifact paths automatically against workdir.
- Let absolute paths escape workdir for referencing external files.
- Share the same workdir across all subworkflows in a run.
- Require no workflow-level declaration — `workdir` is supplied by the CLI.

## Non-Goals

- Per-subworkflow workdir isolation.
- Automatic namespace subdirectories per workflow ID.
- Validation that prompts write only inside workdir.

## Design

### CLI

```sh
daiag run --workflow .daiag/workflows/write_poem/write_poem.star --workdir /output/run1
```

`--workdir` is a required flag when any workflow or subworkflow uses relative
artifact paths or calls `workdir()`. Missing `--workdir` with a relative path
is a validation error.

### `workdir()` Builtin

A zero-argument DSL builtin that returns the workdir path as a string.

```python
poem_path = format("{workdir}/{name}/poem.md", workdir = workdir(), name = name)
```

It may be used anywhere a string expression is accepted:

- `artifact(workdir())`
- `format("{workdir}/...", workdir = workdir())`
- `template_file("...", vars = {"OUT": workdir()})`

### Artifact Path Resolution

`artifact(path)` resolves its path argument at execution time using these rules:

| Path form | Resolution |
|---|---|
| Relative string (`"poem.md"`, `"drafts/v1.md"`) | Prepend workdir: `$workdir/poem.md` |
| `format(...)` result that is relative | Prepend workdir |
| `workdir()` or `format` containing `workdir()` | Already absolute after substitution — use as-is |
| Absolute string (`"/data/spec.md"`) | Use as-is, unchanged |

The rule is simple: if the resolved path starts with `/`, use it as-is.
Otherwise, prepend `--workdir`.

### `workdir()` in Subworkflows

All subworkflows in a run share the same workdir. The runtime passes `workdir`
down implicitly — no workflow needs to declare or forward it as an input.

Example parent:

```python
name = input("name")

wf = workflow(
    id = "feature_development",
    inputs = ["name"],
    steps = [
        subworkflow(
            id = "spec_refinement",
            workflow = "../spec_refinement/spec_refinement.star",
            inputs = {"name": name},
        ),
    ],
)
```

Example child (`spec_refinement.star`) — no `workdir` input needed:

```python
name = input("name")
spec_path = format("{workdir}/{name}/spec.md", workdir = workdir(), name = name)

wf = workflow(
    id = "spec_refinement",
    inputs = ["name"],
    steps = [...],
    output_artifacts = {"spec": spec_path},
)
```

### Referencing Files Outside Workdir

Use an absolute path anywhere a string expression is accepted:

```python
# Read a spec that already exists outside the run output
artifacts = {"spec": artifact("/repos/myproject/docs/spec.md")}
```

Absolute paths are never modified by the runtime.

## Validation Rules

- If `--workdir` is not supplied and any artifact path is relative, fail at
  load time with a clear error.
- If `workdir()` is called and `--workdir` is not supplied, fail at load time.
- `--workdir` must be an absolute path.

## Implementation Tasks

1. Add `workdir()` to the set of predeclared DSL builtins in `internal/starlarkdsl`.
2. Thread `workdir` through the runtime execution context.
3. Resolve artifact paths at execution time: prepend workdir to relative paths.
4. Add `--workdir` validation at workflow load time (fail if missing and needed).
5. Update `docs/workflow-language.md` to document `workdir()` and the resolution rules.
