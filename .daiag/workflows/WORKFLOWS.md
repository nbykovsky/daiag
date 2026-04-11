# Workflow Index

This file is the authoritative index of available workflows in the `daiag` project.

Update this file whenever a workflow is created or modified.
The `workflow-author` agent reads this file to discover available workflows and their input/output contracts.

---

## write_poem

Writes a 10-line poem to a caller-supplied path and exposes the file as a reusable subworkflow output.

File: `.daiag/workflows/write_poem/write_poem.star`

Inputs:
- `poem_path` — absolute or relative path where the poem file should be written

Output Artifacts:
- `poem` — the poem file at `poem_path`

Output Results: none
