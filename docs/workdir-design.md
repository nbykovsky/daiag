# Workdir Design

## Purpose

This document specifies the `workdir` and `projectdir()` features for `daiag`.

`workdir` is the root directory for workflow output artifacts and executor CWD.
`projectdir()` is the project root — the parent of the `.daiag/` directory.
They are separate concepts and must not be conflated.

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
- The CLI silently defaults missing `--workdir` to `os.Getwd()`, making the
  behavior unpredictable.
- Artifact paths are stored as authored strings and only joined with workdir
  for existence checks, causing divergence between stored paths and actual
  file locations in `path_ref(...)`, prompt variables, and logs.

## Separation of Concerns

Three distinct roots exist in a workflow run. They must be kept separate:

| Root | Source | Purpose |
|---|---|---|
| `workdir` | `--workdir` CLI flag | Artifact output root; executor CWD |
| `projectdir` | parent of `.daiag/` | Project source files; module loading base |
| Workflow base dir | location of entry `.star` file | `load(...)` and `subworkflow(...)` path resolution |

`workdir` must never be used as the module loading base or workflow path
validation root. The current runner passes workdir as `BaseDir` for module
loading — this must be corrected as part of this implementation.

## Goals

- Provide `workdir()` as a symbolic DSL expression resolved at execution time.
- Provide `projectdir()` as a load-time DSL builtin resolved from the `.star`
  file location.
- Resolve relative artifact paths against workdir at execution time, producing
  absolute paths stored and used everywhere.
- Apply the same resolution to both `artifact(path)` and `workflow(output_artifacts=...)`.
- Require `--workdir` explicitly — no silent fallback to `os.Getwd()`.
- Share the same workdir across all subworkflows in a run implicitly.

## Non-Goals

- Hard containment enforcement — workdir is an output default, not a sandbox.
- Per-subworkflow workdir isolation.
- Automatic namespace subdirectories per workflow ID.

## Design

### CLI

```sh
daiag run --workflow write_poem --workdir /output/run1
```

`--workdir` is always required. If omitted, the CLI exits with an error before
loading the workflow. There is no silent fallback to `os.Getwd()`.

`--workdir` must be an absolute path.

### `workdir()` — Symbolic Runtime Expression

`workdir()` is not a load-time constant. It is a new symbolic expression type
that carries its value through the expression tree and is resolved to a string
only at execution time, after `--workdir` is known.

This requires a new expression type in `internal/starlarkdsl/builtins.go`
alongside the existing `literalExpr`, `formatExpr`, `pathRefExpr`, and
`inputExpr`. The `unpackStringExpr` and validation functions must be extended
to handle it.

`workdir()` may be used anywhere a string expression is accepted:

```python
poem_path = format("{workdir}/{name}/poem.md", workdir = workdir(), name = name)
artifacts = {"poem": artifact(poem_path)}
```

`workdir()` is not resolved during Starlark evaluation. Its value is injected
by the runtime execution context.

### `projectdir()` — Load-Time Builtin

`projectdir()` returns the project root as a real Starlark string at load time.
It is resolved by walking up from the entry `.star` file's directory until a
directory containing `.daiag/` is found.

```python
prd_path = format("{p}/docs/features/{name}/prd.md", p = projectdir(), name = name)
```

Failure rules:

- If no `.daiag/` ancestor is found after reaching the filesystem root, fail
  at load time with a clear error naming the file and the search path walked.
- `projectdir()` is not affected by `--workdir` or `--workflows-lib`.
- Workflows loaded from a custom `--workflows-lib` outside the project must
  either live under a `.daiag/` ancestor or avoid calling `projectdir()`.

### Artifact Path Resolution

Resolution applies to both `artifact(path)` in tasks and string values in
`workflow(output_artifacts = {...})`.

At execution time, each artifact path is resolved to an absolute path using
this rule:

| Path form | Resolution |
|---|---|
| Relative string (`"poem.md"`, `"drafts/v1.md"`) | `filepath.Join(workdir, path)` |
| `format(...)` result that is relative | `filepath.Join(workdir, resolved)` |
| Contains `workdir()` | already absolute after substitution — use as-is |
| Absolute string (`"/data/spec.md"`) | use as-is, unchanged |

The rule: if the resolved string is not absolute, prepend workdir.

**Resolved paths are absolute.** The absolute path is what gets stored in the
runtime state, passed through `path_ref(...)`, injected into prompt template
variables, written to logs, and returned in child `output_artifacts`.

Note: relative paths containing `../` are the caller's responsibility. The
runtime prepends workdir and uses the result as-is — it does not validate that
the joined path remains under workdir.

### Absolute Artifact Paths as Escape Hatch

Absolute artifact paths deliberately bypass workdir. This supports referencing
files outside the run output, such as in-place edits to project source:

```python
# Edits a file in the project directly
artifacts = {"spec": artifact(format("{p}/docs/spec.md", p = projectdir()))}
```

This is intentional. `workdir` is an output default, not a containment sandbox.

### `workdir()` in Subworkflows

All subworkflows in a run share the same workdir. The runtime injects it
implicitly — no workflow needs to declare or forward it as an input.

The executor CWD for every task (Codex, Claude) is set to `workdir`. This is
separate from `projectdir()`, which reflects the project source root.

Example parent:

```python
name = input("name")

wf = workflow(
    id = "feature_development",
    inputs = ["name"],
    steps = [
        subworkflow(
            id = "spec_refinement",
            workflow = "spec_refinement",
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

The `spec_path` value in `output_artifacts` is resolved to an absolute path at
execution time, so the parent's `path_ref("spec_refinement", "spec")` receives
the correct absolute path.

## Validation Rules

- `--workdir` is required. Missing `--workdir` is a startup error before
  workflow loading begins.
- `--workdir` must be an absolute path.
- `projectdir()` called in a workflow with no `.daiag/` ancestor is a load
  time error.
- Artifact path resolution that produces a non-absolute path after joining
  with workdir is a runtime error (guards against empty workdir).

## Implementation Tasks

1. **Separate module loading base from workdir** — fix `internal/cli/default_runner.go`
   and `internal/starlarkdsl/modules.go` to use the workflow file's directory
   (or project root) as `BaseDir`, not `--workdir`.
2. **Add `workdir()` symbolic expression type** — new type in
   `internal/starlarkdsl/builtins.go`; extend `unpackStringExpr`, validation,
   and `format(...)` handling to accept it; resolve to string in the runtime
   execution context.
3. **Add `projectdir()` load-time builtin** — walk up from the entry `.star`
   file to find the `.daiag/` parent; fail with a clear error if not found.
4. **Require `--workdir`** — remove the `os.Getwd()` fallback in
   `internal/cli/default_runner.go:65`; fail fast if the flag is absent.
5. **Resolve artifact paths to absolute at execution time** — apply resolution
   in `internal/runtime/engine.go` for both `artifact(path)` in tasks and
   string values in `workflow(output_artifacts = {...})`; store and propagate
   only absolute paths.
6. **Set executor CWD to workdir** — confirm `internal/executor/codex/executor.go`
   and `internal/executor/claude/executor.go` use `workdir` as CWD, not
   projectdir or workflow base dir.
7. **Update `docs/workflow-language.md`** — document `workdir()`,
   `projectdir()`, resolution rules, and the `--workdir` requirement.
