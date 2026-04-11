# Agent Guidelines

## Purpose

This repository hosts `daiag`, a Go CLI tool for orchestrating AI agents through
Starlark-defined workflows.
Workflows coordinate prompt files, executor selection, artifact paths, and
review loops over real workspace files.
Use `examples/poem/workflows/poem.star` as the main v1 workflow example when
you need to understand the intended shape of the DSL or runner behavior.
The codebase should stay small, explicit, and easy to change.

## General Principles

- Prefer simple code over flexible abstractions.
- Keep the first implementation narrow and local to the current requirements.
- Add a new abstraction only when at least two concrete uses justify it.
- Favor composition of small packages over large framework-style structures.
- Keep side effects explicit.

## Go Standards

- Use the standard library unless a dependency clearly removes real complexity.
- Keep packages focused and cohesive.
- Prefer concrete types over interfaces at package boundaries unless multiple implementations are already needed.
- Return errors with context using `fmt.Errorf("...: %w", err)`.
- Avoid panics outside unrecoverable startup bugs.
- Pass `context.Context` through execution paths that perform I/O or external process execution.
- Keep exported APIs minimal.
- Use table-driven tests where they improve coverage and readability.
- Keep files short enough to scan without excessive scrolling.

## Project Structure

- `cmd/daiag` should contain the CLI entrypoint only.
- `internal/cli` should handle argument parsing and command dispatch.
- `internal/starlarkdsl` should load and validate workflow definitions.
- `internal/runtime` should execute workflows and manage state.
- `internal/executor/...` should isolate Codex and Claude backend behavior.
- `internal/logging` should format progress output.

## CLI Best Practices

- Keep the CLI predictable and script-friendly.
- Use explicit flags instead of implicit behavior.
- Print concise progress messages to stdout during normal execution.
- Print actionable error messages to stderr.
- Return non-zero exit codes on validation or execution failure.
- Do not print noisy debug details unless the user asks via a flag such as `--verbose`.
- Keep help text short and concrete.

## Workflow Runner Conventions

- Use Starlark as the workflow DSL for v1.
- Treat workflow tasks as explicit orchestration of AI agents driven by prompt
  files and backend executors such as Codex and Claude.
- When the user asks to create or modify a workflow or task, read
  `.daiag/skills/workflow-author/SKILL.md` first and follow it before
  exploring other workflow, example, or implementation files.
- For workflow authoring requests, ask the skill-required clarifying
  questions before inspecting existing workflow files unless the user explicitly
  asks for repository exploration or provides the exact target files to edit.
- Keep task semantics explicit: prompt, executor, artifacts, and result keys.
- Treat artifact paths as declared outputs, not a sandbox.
- Resolve downstream dependencies through explicit references such as path refs and JSON refs.
- Keep execution sequential in v1.
- Prefer path-driven prompts over prompts that derive paths from names.
- When in doubt about workflow structure, follow
  `examples/development-workflow/workflows/feature_development.star` before introducing a
  new pattern.

## Testing

- Add unit tests for parsing, validation, and runtime behavior as packages appear.
- Prefer fast tests that do not require network access.
- Stub or fake executor integrations in unit tests.
- Add integration tests for real CLI behavior only when the command surface is stable.
- Every task should leave the repository in a buildable and testable state.

## Build and Quality Checks

After each meaningful task:

- run `gofmt` on changed Go files
- run package-level tests affected by the change
- run a build of the CLI entrypoint before committing when Go code changed

If a task changes only documentation or repository metadata, document that build and test are not applicable.

## Commits

- Make small commits aligned to one task or one coherent change.
- Use imperative commit messages.
- Do not combine setup, refactor, and feature work in one commit unless they are inseparable.
- Keep the main branch working after every commit.
