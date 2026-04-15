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

Reviews an existing daiag workflow's structure, correctness, and prompt quality, writing a Markdown report to the run directory.

File: `.daiag/workflows/workflow_reviewer/workflow.star`

Inputs:
- `workflow_id` — ID of the workflow to review (must exist under `workflows_lib`)
- `workflows_lib` — absolute path to the workflow catalog directory
- `report_name` — filename for the output review report (e.g. `review.md`)

Output Artifacts:
- `report` — Markdown review report written to the run directory

Output Results: `report_path`

## workflow_patcher

Applies actionable suggestions from a workflow_reviewer report to the target workflow's files in place.

File: `.daiag/workflows/workflow_patcher/workflow.star`

Inputs:
- `report_path` — absolute path to the Markdown review report produced by workflow_reviewer
- `workflow_id` — ID of the workflow to patch (must exist under `workflows_lib`)
- `workflows_lib` — absolute path to the workflow catalog directory

Output Artifacts:
_(none — all edits are made in-place to the existing workflow files)_

Output Results: `outcome`

## workflow_lifecycle

Bootstraps a new workflow from a description, then runs review and patch in a loop (max 3 iterations) until the workflow is clean.

File: `.daiag/workflows/workflow_lifecycle/workflow.star`

Inputs:
- `description` — natural-language description of the workflow to create
- `workflows_lib` — absolute path to the target workflow catalog

Output Artifacts:
- `last_report` — the final review report from the last review iteration

Output Results: `workflow_id`, `workflow_path`, `outcome`

## workflow_composer

Ensures all required building-block workflows exist in the catalog, then assembles the final workflow from a natural-language description.

File: `.daiag/workflows/workflow_composer/workflow.star`

Inputs:
- `description` — natural-language description of the workflow to create
- `workflows_lib` — absolute path to the target workflow catalog

Output Artifacts:
_(none — all artifacts are produced inside nested subworkflow runs)_

Output Results: `workflow_id`, `workflow_path`, `outcome`
