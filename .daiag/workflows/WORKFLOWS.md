# Workflow Index

This file is the authoritative index of available workflows in the `daiag` project.

Update this file whenever a workflow is created or modified.
The `workflow-author` agent reads this file to discover available workflows and their input/output contracts.


## workflow_bootstrapper

Generates a new workflow from a natural-language description in a single step: plans, implements, and registers the workflow in the catalog.

File: `.daiag/workflows/workflow_bootstrapper/workflow.star`

Inputs:
- `description` — natural-language description of the workflow to create
- `workflows_lib` — absolute path to the target workflow catalog

Output Artifacts:
- `blueprint` — planning document written before implementation
- `summary` — authoring summary of created files

Output Results: `workflow_id`, `workflow_path`, `outcome`

## workflow_reviewer

Reviews a daiag workflow's Starlark definition and prompt templates against DSL best practices and writes a concrete improvement report.

File: `/Users/nik/Projects/daiag/.daiag/workflows/workflow_reviewer/workflow.star`

Inputs:
- `workflow_id` — the ID of the workflow to review (must match a subdirectory under `.daiag/workflows/`)

Output Artifacts:
- `review` — markdown review document with findings and verdict

Output Results: `review_path`, `outcome`
