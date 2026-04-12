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

## file_row_grower

Repeatedly adds rows to a file in the existing content style until the line count exceeds a threshold.

File: `.daiag/workflows/file_row_grower/workflow.star`

Inputs:
- `file_name` — path to the file to grow
- `m` — row count threshold; loop exits when line count exceeds this value

Output Artifacts:
- `file` — the grown file at the path given by `file_name`
- `status` — `file_row_grower/count_status.json`

Output Results: `outcome`, `row_count`

## workflow_composer

Turns a high-level workflow description into a formal natural-language workflow blueprint, identifying reusable catalog workflows and documenting any missing capabilities.

File: `.daiag/workflows/workflow_composer/workflow.star`

Inputs:
- `description` — high-level description of the workflow to design

Output Artifacts:
- `blueprint` — `workflow_composer/blueprint.md`

Output Results: `outcome`, `blueprint_path`

## workflow_author_from_blueprint

Reads a natural-language workflow blueprint and implements all missing sub-workflows plus the main orchestrating workflow in the daiag catalog.

File: `.daiag/workflows/workflow_author_from_blueprint/workflow.star`

Inputs:
- `blueprint_path` — path to the natural-language workflow blueprint markdown file

Output Artifacts:
- `summary` — `workflow_author_from_blueprint/summary.md`

Output Results: `workflow_path`, `outcome`
