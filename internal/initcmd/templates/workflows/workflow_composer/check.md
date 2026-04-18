# Check Catalog

Inputs:
- `description`: ${DESCRIPTION}
- `workflows_lib`: ${WORKFLOWS_LIB}
- `check_path`: ${CHECK_PATH}

Instructions:

1. Read `${WORKFLOWS_LIB}/WORKFLOWS.md` to get the current list of available workflows and their capabilities.

2. Analyze `description` to identify all building-block workflows that the target workflow would need to delegate to via `subworkflow(...)`. Focus on discrete, reusable stages — not internal tasks that would be implemented directly in the workflow itself.

3. Check whether each identified building-block workflow already exists in `${WORKFLOWS_LIB}/WORKFLOWS.md`.

4. If any building-block workflow is missing:
   - Set `outcome` to `create`
   - Set `next_description` to a clear, natural-language description of the FIRST missing building-block workflow that should be created. The description must be specific enough that `workflow_lifecycle` can implement it without further clarification. Focus on one workflow at a time.

5. If all identified building-block workflows exist (or no building-block workflows are needed):
   - Set `outcome` to `done`
   - Set `next_description` to an empty string `""`

Outputs:
- Write: ${CHECK_PATH} — a brief analysis listing the identified building blocks, which exist, which are missing, and (when outcome is `create`) which one will be created next
- Return JSON with keys:
  - `outcome`: one of `create`, `done`
  - `next_description`: description of the first missing building-block workflow, or empty string when outcome is `done`

Do not wrap the JSON in Markdown fences.
