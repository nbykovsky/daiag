Workflow goal:
- Turn a user's high-level workflow description into a natural-language workflow blueprint, then turn that blueprint into a runnable `.star` workflow file.

Starting inputs:
- `description` — the user's high-level description of the workflow to create

Final outputs:
- `blueprint_path` — the natural-language workflow blueprint produced by the first stage
- `workflow_path` — the generated `.star` workflow file produced by the second stage
- `outcome` — value reported by the final stage indicating whether the workflow file was produced

Stages:
1. Compose natural-language workflow blueprint
   - Decision: reuse existing workflow `workflow_composer`
   - Purpose: Turns the user's high-level workflow description into a formal natural-language workflow blueprint that identifies reusable catalog workflows and missing capabilities.
   - Receives: parent/user-provided input `description`
   - Produces: file `workflow_composer/blueprint.md` and values `blueprint_path` and `outcome`
   - Notes: Catalog fit is strong. The catalog also includes `poem_generator`, which writes an n-line poem, and `file_row_grower`, which grows a file until a row-count threshold is exceeded; neither fits this stage.
2. Author `.star` workflow from blueprint
   - Decision: create missing workflow
   - Purpose: Reads the natural-language workflow blueprint and writes the corresponding `.star` workflow file.
   - Receives: file produced by the earlier stage at `workflow_composer/blueprint.md`
   - Produces: generated `.star` workflow file path as `workflow_path` and final `outcome`
   - Notes: No catalog workflow currently converts a natural-language workflow blueprint into a `.star` workflow file. The target workflow ID and exact `.star` path can be generated from the blueprint unless the caller later provides a fixed destination.

Information flow:
- The parent workflow receives `description` from the user and passes it to existing workflow `workflow_composer`.
- `workflow_composer` writes the natural-language blueprint to `workflow_composer/blueprint.md` and reports `blueprint_path`.
- The missing authoring workflow reads the blueprint file produced by `workflow_composer`.
- The missing authoring workflow writes the generated `.star` file and reports `workflow_path` and `outcome`.

Missing workflow briefs:
- `workflow_author_from_blueprint`:
  - Purpose: Convert a natural-language workflow blueprint into a runnable daiag `.star` workflow file.
  - Inputs: blueprint file path from an earlier stage, plus optional destination details if the caller needs a fixed workflow ID or output path
  - Outputs: generated `.star` workflow file path and an outcome value such as `complete` or `needs_clarification`
  - Acceptance: The workflow file exists at the reported path, reflects the stages and information flow in the blueprint, and reports `complete` only when no blocking clarification remains.

Open questions:
- none
