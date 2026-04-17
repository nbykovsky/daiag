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

## workflow_assembler

Assembles a complex daiag workflow from catalog components: produces a detailed composition plan, implements the workflow directly (supporting tasks, subworkflows, loops, and conditionals), then runs review and patch in a loop until clean.

File: `.daiag/workflows/workflow_assembler/workflow.star`

Inputs:
- `description` — natural-language description of the workflow to create
- `workflows_lib` — absolute path to the target workflow catalog

Output Artifacts:
- `composition_plan` — Markdown plan documenting which catalog workflows are used as steps and how data flows between them
- `last_report` — final review report from the last review iteration

Output Results: `workflow_id`, `workflow_path`, `outcome`

## code_review_pipeline

Reviews a source file against coding standards, then loops up to 3 times applying fixes and re-reviewing until the file is approved or iterations are exhausted.

File: `.daiag/workflows/code_review_pipeline/workflow.star`

Inputs:
- `file_path` — absolute path to the source file to review
- `standards` — natural-language description of the coding standards to enforce

Output Artifacts:
- `violations_report` — final violations report after all review/fix iterations

Output Results: `outcome`, `violation_count`

## ensure_code_standards

Ensures `docs/code-standards.md` exists and reflects the actual codebase by analyzing the project at runtime and creating or updating the file as needed.

File: `.daiag/workflows/ensure_code_standards/workflow.star`

Inputs: _(none — all paths resolved from `projectdir()` and `run_dir()` at runtime)_

Output Artifacts:
- `analysis` — structured analysis of the project written to the run directory

Output Results: `action`
