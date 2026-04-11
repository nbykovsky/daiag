# Development Workflow Example

This example sketches how a feature-development workflow can fit into the current
`daiag` runner.

It is inspired by the agent-driven process described in:

- `/Users/nik/Projects/ib-broker-trading/docs/research/20260405_1349_development-workflow.md`

## Design Choice

The original workflow is highly expanded and contains several internal control loops:

- spec review and address iterations
- task-by-task implementation
- code review and address iterations
- QA execution, triage, and repair iterations

The current `daiag` runtime can execute:

- sequential `task` steps
- `repeat_until` loops with a fixed body

It does **not** yet support:

- conditional execution inside a loop body
- branching such as "run code addresser only if triage found code issues"
- dynamic task fan-out from a generated task index
- pause-and-resume states that wait for user decisions

Because of that, this example mixes two styles:

- the spec-refinement stage is modeled directly with `repeat_until`
- later implementation and QA stages stay coarse-grained

## What This Example Shows

- a path-driven Starlark workflow for a feature folder
- one task per major workflow stage after spec approval
- an explicit `repeat_until` loop for spec review and address passes
- explicit executor choice per stage
- stable artifacts for the canonical spec and task index
- per-iteration review and status artifacts during spec refinement

## Workflow File

- `examples/development-workflow/workflows/feature-development/feature-development.star`

## Agents

- `examples/development-workflow/agents/spec-writer.md`
- `examples/development-workflow/agents/requirements-reviewer.md`
- `examples/development-workflow/agents/review-addresser.md`
- `examples/development-workflow/agents/qa-test-writer.md`
- `examples/development-workflow/agents/spec-task-splitter.md`
- `examples/development-workflow/agents/task-batch-executor.md`
- `examples/development-workflow/agents/code-refiner.md`
- `examples/development-workflow/agents/qa-refiner.md`
- `examples/development-workflow/agents/docs-updater.md`

## Sample Input

- `examples/development-workflow/docs/features/indicators/prd.md`

## Intended Outputs

For feature `indicators`, the workflow writes into:

- `examples/development-workflow/docs/features/indicators/spec.md`
- `examples/development-workflow/docs/features/indicators/spec_review_1.md`
- `examples/development-workflow/docs/features/indicators/spec_refine_status_1.md`
- `examples/development-workflow/docs/features/indicators/qa_tests.md`
- `examples/development-workflow/docs/features/indicators/tasks.md`
- `examples/development-workflow/docs/features/indicators/task_execution_status.md`
- `examples/development-workflow/docs/features/indicators/code_review_status.md`
- `examples/development-workflow/docs/features/indicators/qa_status.md`
- `examples/development-workflow/docs/features/indicators/docs_update_status.md`

If refinement needs more than one pass, the workflow also creates:

- `examples/development-workflow/docs/features/indicators/spec_review_<n>.md`
- `examples/development-workflow/docs/features/indicators/spec_refine_status_<n>.md`

## Important Note

This example is a **supported approximation** of the target workflow, not a full
one-to-one encoding of the original `ib-broker-trading` process.

The spec loop is the closest part to the source workflow:

- `requirements-reviewer` writes a review report with one of `ready`, `ready_with_concerns`, or `not_ready`
- `review-addresser` always runs, but it becomes a no-op when the review is already `ready` or `not_ready`
- `review-addresser` edits the spec only when the report says `ready_with_concerns`
- `repeat_until` terminates when `review-addresser` returns `loop_outcome = stop`

The fully expanded version would want additional runtime features such as:

- conditional branches inside loops
- dynamic iteration over generated task files
- richer workflow status outcomes such as `blocked`
