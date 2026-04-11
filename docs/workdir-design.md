# Workdir Design

## Purpose

This document specifies the `workdir` and `projectdir()` features for `daiag`.

`workdir` is the root directory for workflow output artifacts and executor CWD.
`projectdir()` returns the project root for the calling module.
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

Four distinct directory concepts exist in a workflow run:

| Concept | Source | Purpose |
|---|---|---|
| `workdir` | `--workdir` CLI flag | Artifact output root; executor CWD |
| `projectdir` | parent of `.daiag/` relative to the calling `.star` module | Path value for reading project source files |
| Module resolution dir | directory of the importing `.star` module | Starting point for resolving relative `load(...)` paths |
| Module allowed boundary | workflows library root | Upper bound for module path validation — load paths must stay under this |

`projectdir` is a plain path value available to workflows. It is not the
module loading base and not the allowed boundary.

`workdir` must never be used as the module resolution directory or allowed
boundary. The current runner passes `workdir` as `BaseDir` for module
loading — this must be corrected as part of this implementation.

## Goals

- Provide `workdir()` as a symbolic DSL expression resolved at execution time.
- Provide `projectdir()` as a load-time DSL builtin resolved from the calling
  `.star` module's location.
- Resolve relative `artifact(path)` and `output_artifacts` values against
  workdir at execution time, producing absolute paths stored and used
  everywhere.
- Require `--workdir` explicitly — no silent fallback to `os.Getwd()`.
- Create `workdir` on disk if it does not exist.
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

`--workdir` must be an absolute path. The CLI creates it if it does not exist
(`mkdir -p`) before workflow execution begins, since it is the executor CWD.

### `workdir()` — Symbolic Runtime Expression

`workdir()` is not a load-time constant. It is a new symbolic expression type
that carries its value through the expression tree and is resolved to a string
only at execution time, after `--workdir` is known.

This requires:
- A new model type in `internal/workflow/model.go` (alongside `Literal`,
  `FormatExpr`, etc.)
- A corresponding Starlark value in `internal/starlarkdsl/values.go`
- A builtin registration in `internal/starlarkdsl/builtins.go`
- Extension of validation and `unpackStringExpr` to accept the new type
- A runtime resolver that substitutes the concrete workdir string during
  execution in `internal/runtime/engine.go`

`workdir()` may be used anywhere a string expression is accepted:

```python
poem_path = format("{workdir}/{name}/poem.md", workdir = workdir(), name = name)
artifacts = {"poem": artifact(poem_path)}
```

### `projectdir()` — Load-Time Builtin

`projectdir()` returns the project root as a real Starlark string at load time.
It is resolved per calling module: walk up from the directory of the `.star`
file where `projectdir()` is evaluated until a directory containing `.daiag/`
is found.

```python
prd_path = format("{p}/docs/features/{name}/prd.md", p = projectdir(), name = name)
```

Using the calling module's directory (not the entry file's directory) means
each loaded module resolves projectdir relative to itself, consistent with how
`template_file(...)` resolves paths.

Failure rules:

- If no `.daiag/` ancestor is found after reaching the filesystem root, fail
  at load time with a clear error naming the calling module and the path walked.
- Workflows that live outside any `.daiag/` project must not call
  `projectdir()`. The error message should suggest passing the project path as
  an explicit workflow input instead.

### Artifact Path Resolution

Resolution applies to:
- `artifact(path)` values declared in tasks
- String values in `workflow(output_artifacts = {...})`

It does **not** apply to arbitrary string expressions in `template_file(vars = {...})`.
Prompt variables like `"POEM_PATH": "poem.md"` render as authored. If a prompt
variable needs a workdir-rooted path, the workflow must use `workdir()` explicitly:

```python
prompt = template_file("write_poem.md", vars = {
    "POEM_PATH": format("{workdir}/poem.md", workdir = workdir()),
})
```

Resolution rule at execution time:

| Path form | Resolution |
|---|---|
| Relative string (`"poem.md"`, `"drafts/v1.md"`) | `filepath.Join(workdir, path)` |
| `format(...)` result that is relative | `filepath.Join(workdir, resolved)` |
| Contains `workdir()` | already absolute after substitution — use as-is |
| Absolute string (`"/data/spec.md"`) | use as-is, unchanged |

The rule: if the resolved string is not absolute, prepend workdir.

**Resolved paths are absolute.** The absolute path is what gets stored in
runtime state, passed through `path_ref(...)`, injected into prompt template
artifact variables when they reference an artifact path, written to logs, and
returned in child `output_artifacts`.

Note: relative paths containing `../` are the caller's responsibility. The
runtime prepends workdir and uses the result as-is.

### Absolute Artifact Paths and Executor Access

Absolute artifact paths deliberately bypass workdir. This supports in-place
edits to project source files:

```python
artifacts = {"spec": artifact(format("{p}/docs/spec.md", p = projectdir()))}
```

This is intentional. `workdir` is an output default, not a containment sandbox.

**Known limitation:** executors currently run with workdir as their CWD and
access root. Codex uses `-C req.Workdir` with workspace-write scoped to
workdir; Claude uses `--add-dir req.Workdir`. If an artifact path points
outside workdir (e.g., to projectdir), the executor may not have write access
to that location.

For v1, this is the caller's responsibility. A future extension may add
projectdir as an additional executor access root when absolute artifact paths
are declared outside workdir.

### `workdir()` in Subworkflows

All subworkflows in a run share the same workdir. The runtime injects it
implicitly — no workflow needs to declare or forward it as an input.

The executor CWD for every task (Codex, Claude) is set to `workdir`.

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
- `projectdir()` called in a module with no `.daiag/` ancestor is a load
  time error.

## Implementation Tasks

1. **Separate module loading base from workdir** — fix `internal/cli/default_runner.go`
   and `internal/starlarkdsl/modules.go` to use the calling module's directory
   as the module resolution directory and the entry workflow directory as the
   allowed boundary; never use `--workdir` for either.

2. **Add `workdir()` symbolic expression** — add a new model type to
   `internal/workflow/model.go`; add a corresponding Starlark value to
   `internal/starlarkdsl/values.go`; register the builtin in
   `internal/starlarkdsl/builtins.go`; extend `unpackStringExpr` and
   validation; resolve to the concrete workdir string in
   `internal/runtime/engine.go`.

3. **Add `projectdir()` load-time builtin** — register in
   `internal/starlarkdsl/builtins.go`; walk up from the calling module's
   directory to find the `.daiag/` parent; fail with a clear error if not
   found.

4. **Require `--workdir` and create it** — remove the `os.Getwd()` fallback
   in `internal/cli/default_runner.go:65`; fail fast if the flag is absent;
   reject relative paths with an explicit error rather than converting with
   `filepath.Abs`; `mkdir -p` the workdir before execution begins.

5. **Resolve artifact paths to absolute at execution time** — apply resolution
   in `internal/runtime/engine.go` for both `artifact(path)` in tasks and
   string values in `workflow(output_artifacts = {...})`; store and propagate
   only absolute paths.

6. **Set executor CWD to workdir** — confirm `internal/executor/codex/executor.go`
   and `internal/executor/claude/executor.go` use `--workdir` as CWD and
   access root; document that absolute artifact paths outside workdir depend
   on executor permissions.

7. **Update `docs/workflow-language.md`** — document `workdir()`,
   `projectdir()`, resolution rules, the `--workdir` requirement, and the
   executor access limitation for absolute paths.
