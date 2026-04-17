# Code Standards

This document captures the conventions observed in and expected of this codebase.
Existing code may not yet comply with every rule listed here.

---

## Go

### Naming

- **Packages**: lowercase single word matching the directory name (`cli`, `runtime`, `starlarkdsl`, `workflow`, `logging`, `executor`, `claude`, `codex`).
- **Exported types**: PascalCase (`Engine`, `RunInput`, `RunResult`, `TaskRequest`).
- **Unexported types**: camelCase (`state`, `stepError`).
- **Exported functions and methods**: PascalCase (`New`, `Run`, `Load`).
- **Unexported functions and methods**: camelCase (`runNodes`, `renderPrompt`, `resolveArtifactPath`).
- **Variables**: camelCase (`baseDir`, `absPath`, `childInputs`).
- **Config structs**: `XxxConfig` suffix (`RunConfig`, `ValidateConfig`, `BootstrapConfig`).
- **Test doubles**: `fakeXxx` prefix (`fakeRunner`, `fakeExecutor`).
- **Constants**: camelCase for unexported inline constants (`usageText`).
- **Error variables**: constructed at the point of failure — no package-level sentinel errors.

### File Names

- Go source files: `snake_case.go` (`loader_test.go`, `workflow_ref.go`, `default_runner.go`).
- Test files: `xxx_test.go` adjacent to the file under test.

### Package Structure

```
daiag/
  cmd/daiag/main.go       # CLI entrypoint only — no logic
  internal/
    cli/                  # argument parsing and command dispatch
    runtime/              # workflow execution engine, state management
    starlarkdsl/          # Starlark DSL loading and builtins
    workflow/             # domain model: pure data types and validation only
    executor/
      claude/             # Claude executor backend
      codex/              # Codex executor backend
      subprocess.go       # shared subprocess abstraction
    logging/              # progress output formatting
```

Rules:
- All application code lives under `internal/` — there is no public API surface.
- `workflow/` holds domain model types only; no I/O or execution logic belongs there.
- `runtime/` depends on `workflow/` and `logging/`; executors are injected via interface.
- `starlarkdsl/` depends on `workflow/` and `go.starlark.net` only.
- Each package has a single, focused responsibility. Do not merge concerns.

### Error Handling

- Wrap all errors with context: `fmt.Errorf("short action phrase: %w", err)`.
- Error message prefixes are lowercase, action-oriented: `"resolve project dir"`, `"read prompt template"`, `"load subworkflow"`.
- Return errors from all fallible functions. Do not swallow errors silently.
- Panics are reserved for unrecoverable startup conditions only.
- Use `errors.Is` / `errors.As` for error inspection. When `errors.As` cannot be used (e.g., non-pointer receiver on a custom type), walk the unwrap chain manually.
- The `stepError` pattern — a struct carrying `(StepID, Err)` that implements `Error()` and `Unwrap()` — is the approved way to attach step identity to an error as it propagates up the call stack.

### Context

- Pass `context.Context` as the first argument through all functions that perform I/O or launch external processes.
- Do not store `context.Context` in structs.

### Dependencies

- Use the standard library unless an external package clearly removes real complexity.
- The only approved external dependency is `go.starlark.net`. Add new dependencies only after explicit discussion.

### Abstractions

- Add an interface or abstraction only when at least two concrete implementations already exist and justify it.
- Prefer concrete types at package boundaries unless multiple implementations are required.
- Three similar lines of code is better than a premature abstraction.

### Formatting and Quality

- Run `gofmt` on all changed Go files before committing.
- Run all package-level tests affected by a change before committing.
- Build the CLI entrypoint (`cmd/daiag`) and confirm it compiles before committing Go changes.

---

## Testing

- Tests live in the same package as the code under test (`package cli`, `package runtime`, etc.). Do not use a separate `_test` package suffix unless testing the exported API from the outside.
- Test function names: `TestXxxYyy` (PascalCase matching the function or scenario, e.g., `TestEngineRunWorkflowWithLoop`).
- Use table-driven tests where multiple cases add coverage without excessive boilerplate.
- Test doubles are lightweight structs named `fakeXxx` that implement the required interface locally.
- Use `t.TempDir()` for temporary file fixtures. Write fixture files via a local `writeFile(t, ...)` helper rather than inline `os` calls.
- Make injectable any field that produces non-deterministic output (e.g., a `Now` function on a logger).
- Tests must not hit the network or invoke real executors. Fake all I/O at the boundary.

---

## CLI

- Progress output goes to stdout; errors go to stderr.
- Exit codes: `0` = success, `1` = execution error, `2` = usage or validation error.
- Error lines printed by the CLI use the prefix `"error: "`.
- Flag names are consistent across commands: `--workflow`, `--projectdir`, `--run-dir`, `--workflows-lib`.
- `--input key=value` is the repeatable flag for workflow inputs.
- `--verbose` enables debug details; suppress debug output by default.
- Keep help text short and concrete. Avoid flags with implicit or surprising behavior.

---

## Starlark Workflow DSL

### File Layout

- Workflows live at `.daiag/workflows/<name>/workflow.star` with sibling prompt `.md` files.
- Example workflows live under `examples/<name>/workflows/`.
- Workflow file names: `snake_case.star` or `kebab-case.star`.
- Prompt template files: `snake_case.md` or `kebab-case.md`.

### Naming

- Task IDs: `snake_case` (`write_poem`, `review_poem`, `extend_until_ready`).
- Variable names: `snake_case` (`feature_dir`, `poem_path`, `default_executor`).
- Built-in DSL names use `snake_case` and are fixed: `task`, `repeat_until`, `path_ref`, `json_ref`, `loop_iter`, `template_file`, `run_dir`, `projectdir`, `param`, `input`.

### Task Conventions

- Every task declares its `prompt`, `executor`, `output_artifacts`, and `output_results` explicitly.
- Use `path_ref(step_id, key)` and `json_ref(step_id, field)` to wire downstream dependencies through explicit references — do not derive paths from names.
- Use `param()` for workflow-level parameters. `param()` is disabled inside subworkflow context.
- Prompt templates use `${VAR_NAME}` placeholder syntax rendered at runtime.
- Keep execution sequential in v1. Do not introduce parallel task execution without a design decision.
- Treat artifact paths as declared outputs, not a general-purpose sandbox.

### Subworkflows

- Subworkflow cycle detection runs at load time. Cycles are an error.
- Use `examples/development-workflow/workflows/feature-development.star` as the reference pattern when introducing new workflow structure.

---

## Markdown and Prompt Templates

- Prompt files are plain Markdown; use `${VAR_NAME}` placeholders for runtime substitution.
- Documentation files live under `docs/`.
- Agent definitions live under `.daiag/agents/` or `examples/*/agents/`.
- Do not embed logic in prompt files — keep them declarative descriptions of what the agent should do.

---

## Commits

- One commit per task or coherent change. Do not bundle setup, refactor, and feature work unless they are inseparable.
- Commit messages use the imperative mood: `Add`, `Fix`, `Remove`, `Refactor`.
- The main branch must build and pass tests after every commit.
