# Implement Workflow from Blueprint

Inputs:
- `blueprint_path`: ${BLUEPRINT_PATH}
- `summary_path`: ${SUMMARY_PATH}

Instructions:
1. Read the natural-language workflow blueprint at `${BLUEPRINT_PATH}`. The blueprint has these sections:
   - **Workflow goal** — the overall purpose of the workflow being described.
   - **Starting inputs** — the runtime inputs the parent workflow accepts.
   - **Final outputs** — the artifacts and result keys the parent workflow exposes.
   - **Stages** — ordered list of steps, each with a Decision (`reuse existing workflow <id>` or `create missing workflow`), Purpose, Receives, Produces, and Notes.
   - **Information flow** — prose description of how data moves between stages.
   - **Missing workflow briefs** — specs for workflows that must be created.
   - **Open questions** — unresolved design questions.
2. If Open questions lists any unresolved items, set `outcome` to `needs_clarification` and write the open questions to `${SUMMARY_PATH}`, then stop.
3. Read `.daiag/workflows/WORKFLOWS.md` to understand the existing workflow catalog. Treat every `## <id>` section as an available workflow that can be used with `subworkflow(...)`.
4. For each entry in **Missing workflow briefs**:
   a. Derive the workflow ID from the entry name (already lowercase with underscores).
   b. Skip if the ID already exists in `.daiag/workflows/WORKFLOWS.md`.
   c. Create the directory `.daiag/workflows/<id>/` if it does not exist.
   d. Write `.daiag/workflows/<id>/workflow.star` following these conventions exactly:
      - Assign `wf = workflow(...)` at the top level.
      - Declare `inputs = [...]` (use `[]` when there are no inputs); obtain values via `input("name")`.
      - Set `default_executor = {"cli": "codex", "model": "gpt-5.4"}` on the workflow.
      - Define tasks inline as `task(...)` values inside `workflow(steps = [...])`.
      - Every task must declare non-empty `artifacts` and `result_keys`.
      - Every artifact value must be wrapped in `artifact(...)`.
      - Use relative artifact paths namespaced under the workflow ID (e.g. `<id>/output.md`).
      - Reference runtime inputs with `input("name")`.
      - Reference artifacts from an earlier step with `path_ref("step_id", "artifact_key")`.
      - Reference result values from an earlier step with `json_ref("step_id", "field")`.
      - Declare both `output_artifacts` and `output_results`; use `{}` when a category is empty.
   e. Write one prompt file `.daiag/workflows/<id>/<task_id>.md` per task. Each prompt must:
      - Start with `# <Task Title>`.
      - Have an `Inputs:` section listing every dynamic value with `${PLACEHOLDER}` syntax.
      - Have an `Instructions:` numbered list stating exactly which files to read and write.
      - Have an `Outputs:` section listing every artifact file and the exact JSON keys to return.
      - End with `Do not wrap the JSON in Markdown fences.`
      - Every `${NAME}` placeholder must have a matching `vars` entry in the corresponding task.
   f. Append an entry for this workflow to `.daiag/workflows/WORKFLOWS.md` using this format:
      ```
      ## <id>

      <one-sentence description>

      File: `.daiag/workflows/<id>/workflow.star`

      Inputs:
      - `<input>` — <description>

      Output Artifacts:
      - `<key>` — `<path>`

      Output Results: `<key1>`, `<key2>`, ...
      ```
5. Determine the main workflow ID from the **Workflow goal** (lowercase, underscores; e.g. `turn a user's high-level workflow description into ...` → derive a short descriptive id).
6. Skip creating the main workflow if its ID already exists in `.daiag/workflows/WORKFLOWS.md`.
7. Create `.daiag/workflows/<main_id>/workflow.star` that composes all stages in order:
   - For each stage with `Decision: reuse existing workflow <id>`, use `subworkflow(id = "<step_id>", workflow = "<id>", inputs = {...})`.
   - For each stage with `Decision: create missing workflow`, use `subworkflow(id = "<step_id>", workflow = "<missing_id>", inputs = {...})`.
   - Wire inter-stage data with `path_ref("step_id", "artifact_key")` and `json_ref("step_id", "field")` as documented in **Information flow**.
   - Declare `inputs` from **Starting inputs** using `input(...)`.
   - Declare `output_artifacts` and `output_results` from **Final outputs**.
8. Append the main workflow entry to `.daiag/workflows/WORKFLOWS.md` using the same format as step 4f.
9. Write a brief summary to `${SUMMARY_PATH}` listing every workflow ID created, its `.star` file path, and its prompt files.

Outputs:
- Write/update: ${SUMMARY_PATH}
- Return JSON with keys:
  - `workflow_path`: path to the main workflow `.star` file (e.g. `.daiag/workflows/<main_id>/workflow.star`)
  - `outcome`: one of `complete`, `needs_clarification`

Do not wrap the JSON in Markdown fences.
