# Examples

This directory contains runnable workflow examples for `daiag`.

## Poem

The poem example shows the main v1 workflow pattern:

- a writer task creates an initial file
- an extender task updates that file
- a reviewer task decides whether the loop should stop
- the workflow repeats until the reviewer returns `ready`
- each review iteration is written to its own file with an iteration suffix

Files:

- workflow: `examples/poem/workflows/poem.star`
- workflow helpers:
  - `examples/poem/workflows/lib/paths.star`
  - `examples/poem/workflows/lib/tasks.star`
- prompts:
  - `examples/poem/agents/poem-writer.md`
  - `examples/poem/agents/poem-extender.md`
  - `examples/poem/agents/poem-reviewer.md`
- input spec: `examples/poem/docs/features/rain/spec.md`

Run it from the repository root:

```sh
go run ./cmd/daiag run --workflow examples/poem/workflows/poem.star --param name=rain
```

Prerequisites:

- `codex` CLI must be installed and authenticated
- `claude` CLI must be installed and authenticated

Expected outputs:

- `examples/poem/docs/features/rain/poem.md`
- `examples/poem/docs/features/rain/review-1.txt`
- `examples/poem/docs/features/rain/review-2.txt`
- additional `review-<n>.txt` files if the loop runs longer

The example workflow uses:

- `codex` with model `gpt-5.4` for writing and extending
- `claude` with model `sonnet` for review

## Development Workflow

The development workflow example is a research-oriented sketch for a much larger
agent pipeline inspired by `ib-broker-trading`.

Files:

- example overview: `examples/development-workflow/README.md`
- workflow: `examples/development-workflow/workflows/feature-development.star`
- agents: `examples/development-workflow/agents/`

Important note:

- this example uses a real `repeat_until` loop for spec review and refinement
- later implementation and QA stages are still intentionally coarse-grained
- the fully expanded workflow from `ib-broker-trading` is not currently supported by `daiag`

Current runner gaps for the fully expanded topology:

- no branch selection based on prior task results
- no dynamic fan-out over generated task files
- no built-in blocked or pause-and-resume workflow state
