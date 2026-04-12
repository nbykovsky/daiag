# Workflow Index

This file is the authoritative index of available workflows in the `daiag` project.

Update this file whenever a workflow is created or modified.
The `workflow-author` agent reads this file to discover available workflows and their input/output contracts.

## poem_generator

Writes a poem with exactly `n` lines to a file.

File: `.daiag/workflows/poem_generator/workflow.star`

Inputs:
- `n` — number of lines the poem should contain

Output Artifacts:
- `poem` — `poem_generator/poem.md`

Output Results: `poem_path`
