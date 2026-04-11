# Subworkflow Implementation Tasks

## Purpose

This document breaks `docs/subworkflow-design.md` into implementation tasks.

Each task must leave the repository buildable and testable.
Do not batch multiple tasks into one commit.

## Per-Task Done Criteria

Every implementation task below must end with:

1. Tests added or updated for the behavior changed by the task.
2. `gofmt` run on changed Go files.
3. Affected package tests run.
4. `go test ./...` run.
5. `go build -o /tmp/daiag-subworkflow-check ./cmd/daiag` run.
6. A commit containing only that task's coherent change.

If a test or build command cannot run because of the environment, document the
exact command and failure before committing.

## Task 1: Add Workflow Inputs

Goal: make `workflow(inputs = [...])` and `input(name)` work for top-level
workflows without adding subworkflow support yet.

Implementation:

- Add `Inputs []string` to `workflow.Workflow`.
- Add `workflow.InputRef` as both `ValueExpr` and `StringExpr`.
- Add Starlark `input(name)` and `inputValue`.
- Extend `workflow(...)` unpacking for optional `inputs`.
- Update `unpackStringExpr` and `unpackValueExpr` for `input(...)`.
- Add `Validator.Inputs map[string]string`.
- Validate unique non-empty workflow input names.
- Validate `input("x")` against the workflow's declared inputs.
- Validate top-level declared inputs against CLI-provided input keys.
- Add `RunInput.Inputs map[string]any` and `state.inputs`.
- Resolve `InputRef` in `resolveStringExpr` and `resolveValueExpr`.
- Add CLI `--input key=value`.
- Keep `--param key=value` as a compatibility alias.
- Reject conflicting `--input name=x` and `--param name=y` values.
- Keep `param(...)` working for existing top-level workflows.

Tests:

- CLI accepts `--input key=value`.
- CLI keeps `--param key=value` compatibility.
- CLI rejects conflicting `--input` and `--param` values for the same key.
- Loader accepts a workflow with declared inputs and `input(...)`.
- Loader rejects duplicate workflow input declarations.
- Loader rejects `input("x")` when `"x"` is not declared.
- Loader rejects a missing top-level input binding.
- Runtime resolves `input(...)` in prompt template vars.
- Runtime resolves `input(...)` inside `format(...)`.
- Runtime resolves `input(...)` in artifact declarations.

Run:

```sh
go test ./internal/cli ./internal/starlarkdsl ./internal/runtime ./internal/workflow
go test ./...
go build -o /tmp/daiag-subworkflow-check ./cmd/daiag
```

Commit:

```sh
git commit -m "Add workflow inputs"
```

## Task 2: Add Workflow Output Contracts

Goal: let workflows declare public outputs without adding subworkflow execution
yet.

Implementation:

- Add `OutputArtifacts map[string]workflow.StringExpr` to `workflow.Workflow`.
- Add `OutputResults map[string]workflow.ValueExpr` to `workflow.Workflow`.
- Extend `workflow(...)` unpacking for optional `output_artifacts` and
  `output_results`.
- Validate output keys as non-empty and unique within each map.
- Validate output artifact expressions as string expressions.
- Validate output result expressions as value expressions.
- Allow output expressions to reference declared `input(...)` values.
- Validate output expressions against the final workflow step scope.
- Preserve current loop behavior: tasks in `repeat_until(max_iters >= 1)` are
  visible to later outputs using latest-successful-execution semantics.
- Refactor validation internals from task-only lookup to a unified `nodeInfo`
  lookup for task artifact keys and result keys.

Tests:

- Loader accepts `output_artifacts` with `path_ref(...)`.
- Loader accepts `output_artifacts` with `input(...)` pass-through.
- Loader accepts `output_results` with `json_ref(...)`.
- Loader rejects output references to unknown steps.
- Loader rejects output references to unknown artifact or result keys.
- Loader accepts output references to tasks inside a `repeat_until(...)` body.
- Existing duplicate-ID validation still works after the `nodeInfo` refactor.

Run:

```sh
go test ./internal/starlarkdsl ./internal/workflow
go test ./...
go build -o /tmp/daiag-subworkflow-check ./cmd/daiag
```

Commit:

```sh
git commit -m "Add workflow output contracts"
```

## Task 3: Load Subworkflow Definitions

Goal: parse and load `subworkflow(...)` nodes, but do not execute them yet.

Implementation:

- Add `workflow.Subworkflow` with `ID`, `WorkflowPath`, `ModuleDir`,
  `Workflow`, and `Inputs`.
- Add Starlark `subworkflow(...)` and `subworkflowValue`.
- Capture `ModuleDir` with `currentCallerModulePath(thread)`.
- Extend `unpackSteps` to accept subworkflow values.
- Resolve subworkflow paths with the same boundary rules as `load(...)`.
- Add a per-call `workflowLoadContext` threaded through recursive workflow
  loads.
- Detect subworkflow file cycles with `workflowLoadContext.loading`.
- Load each subworkflow file through a workflow-file loading path that permits
  top-level `wf`.
- Disable `param(...)` while loading subworkflow files and their helper modules.
- Attach a fresh child `*workflow.Workflow` instance to every
  `workflow.Subworkflow`.
- Do not share mutable workflow model pointers between two subworkflow nodes
  even when they reference the same child file.
- Allow omitted `inputs` on `subworkflow(...)` and treat it as `{}`.

Tests:

- Loader accepts a parent workflow containing a subworkflow node.
- Loader resolves subworkflow paths relative to the Starlark caller module.
- Loader rejects subworkflow paths outside the workflow base directory.
- Loader rejects a child workflow without top-level `wf`.
- Loader rejects `param(...)` in a child workflow.
- Loader rejects a direct subworkflow cycle.
- Loader rejects an indirect subworkflow cycle.
- Loader accepts child workflows with no declared inputs when `inputs` is
  omitted.
- Loader accepts child workflows with no declared inputs when `inputs = {}`.
- Two subworkflow nodes referencing the same child file receive distinct child
  workflow model instances.

Run:

```sh
go test ./internal/starlarkdsl ./internal/workflow
go test ./...
go build -o /tmp/daiag-subworkflow-check ./cmd/daiag
```

Commit:

```sh
git commit -m "Load subworkflow definitions"
```

## Task 4: Validate Subworkflow Boundaries

Goal: make validation scope-aware for subworkflows and register child public
outputs in the parent scope.

Implementation:

- Add the `Subworkflow` case to workflow validation.
- Use one workflow-local `allIDs` set per workflow scope.
- Use one workflow-local `seenNodes map[string]nodeInfo` per workflow scope.
- Keep loop bodies in the current workflow scope.
- Validate each child workflow in a fresh child workflow scope.
- Start child workflow validation with an empty active-loop set.
- Validate parent-side subworkflow input expressions against the parent scope.
- Pass the parent active-loop set when validating parent-side subworkflow input
  expressions.
- Check that every child declared input has a parent binding.
- Reject parent bindings for unknown child inputs.
- Register the subworkflow ID in the parent `seenNodes` map with artifact keys
  from child `OutputArtifacts` and result keys from child `OutputResults`.
- Ensure parent workflows cannot reference child internal task IDs.
- Ensure child workflows cannot reference parent task IDs directly.

Tests:

- Parent can reference child declared artifact output with `path_ref(...)`.
- Parent can reference child declared result output with `json_ref(...)`.
- Parent cannot reference child internal task IDs.
- Child cannot reference parent task IDs directly.
- Parent and child may use the same internal task ID without duplicate-ID
  errors.
- Duplicate IDs inside one child workflow still fail.
- Parent subworkflow binding can use `loop_iter(...)` when the subworkflow node
  is inside that loop.
- Child validation does not inherit the parent's active loop set.
- Missing child input binding fails.
- Unknown child input binding key fails.

Run:

```sh
go test ./internal/starlarkdsl ./internal/workflow
go test ./...
go build -o /tmp/daiag-subworkflow-check ./cmd/daiag
```

Commit:

```sh
git commit -m "Validate subworkflow boundaries"
```

## Task 5: Execute Subworkflows

Goal: run subworkflow nodes and make their declared outputs available to later
parent steps.

Implementation:

- Add the `Subworkflow` case to `runtime.Engine.runNodes`.
- Evaluate subworkflow input expressions against the parent runtime state.
- Run the attached child workflow with a fresh child runtime state.
- Pass child input values through `RunInput.Inputs`.
- Share context, executors, logger, base dir, and workdir.
- Do not share artifacts, results, active loop state, or input map.
- After child completion, evaluate child `OutputArtifacts` and `OutputResults`
  against the child final state.
- Register evaluated child output artifacts under
  `st.artifacts[subworkflow.ID]`.
- Register evaluated child output results under `st.results[subworkflow.ID]`.
- Preserve latest-successful-execution behavior for subworkflows inside parent
  loops.

Tests:

- Parent passes literal input values to a child workflow.
- Parent passes `path_ref(...)` values to a child workflow.
- Parent passes `json_ref(...)` values to a child workflow.
- Child output artifact is visible to later parent tasks with
  `path_ref(subworkflow_id, key)`.
- Child output result is visible to later parent tasks or predicates with
  `json_ref(subworkflow_id, key)`.
- Child output expression can pass through a declared `input(...)` value.
- Child runtime state does not leak internal task artifacts to the parent.
- Child runtime state does not leak internal task results to the parent.
- Subworkflow inside a loop exposes the latest successful execution.

Run:

```sh
go test ./internal/runtime ./internal/starlarkdsl ./internal/workflow
go test ./...
go build -o /tmp/daiag-subworkflow-check ./cmd/daiag
```

Commit:

```sh
git commit -m "Execute subworkflows"
```

## Task 6: Add Subworkflow Failure Reporting And Logging

Goal: make nested failures and progress output understandable.

Implementation:

- Qualify child `stepError` values at every subworkflow boundary.
- Convert child `review_spec` failures into parent `spec.review_spec` failures.
- Apply the same boundary rule recursively so nested failures become paths such
  as `spec.refine.inner_task`.
- Add subworkflow start, done, and failed logging events.
- Decide whether child task log lines are qualified directly or grouped under
  surrounding subworkflow events; prefer qualified IDs if the implementation is
  small.

Tests:

- Single-level child failure reports the subworkflow ID and child step ID.
- Nested child failure reports recursive context such as
  `spec.refine.inner_task`.
- Subworkflow start and done events are logged.
- Subworkflow failure event includes the parent subworkflow ID and child step
  context.

Run:

```sh
go test ./internal/runtime ./internal/logging
go test ./...
go build -o /tmp/daiag-subworkflow-check ./cmd/daiag
```

Commit:

```sh
git commit -m "Report subworkflow failures"
```

## Task 7: Update Examples And Language Docs

Goal: document the implemented language surface and add one real example.

Implementation:

- Update `docs/workflow-language.md` from planned to implemented behavior for:
  - `workflow(inputs = ...)`
  - `workflow(output_artifacts = ...)`
  - `workflow(output_results = ...)`
  - `input(name)`
  - `subworkflow(...)`
  - `--input`
  - `--param` compatibility behavior
- Keep `docs/subworkflow-design.md` as design history or update it to point at
  the language reference.
- Add or refactor an example subworkflow, likely the spec-refinement part of
  `examples/development-workflow`.
- Add any reusable helper files needed by the example.

Tests:

- Loader test covers the new example workflow.
- CLI parsing test covers the documented `--input` command.
- If the example can run with fake executors, add a runtime or integration-style
  test around it.

Run:

```sh
go test ./internal/cli ./internal/starlarkdsl ./internal/runtime
go test ./...
go build -o /tmp/daiag-subworkflow-check ./cmd/daiag
```

Commit:

```sh
git commit -m "Document subworkflow language support"
```
