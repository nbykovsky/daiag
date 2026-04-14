# Apply Review

Inputs:
- `REVIEW_PATH`: ${REVIEW_PATH}
- `WORKFLOW_ID`: ${WORKFLOW_ID}
- `WORKFLOWS_DIR`: ${WORKFLOWS_DIR}
- `CHANGES_PATH`: ${CHANGES_PATH}

Instructions:

1. Read the review document at `${REVIEW_PATH}`. Collect every finding labelled **ISSUE** or **SUGGESTION**.

2. Read `${WORKFLOWS_DIR}/workflow.star` and every `.md` file in `${WORKFLOWS_DIR}/` (these are prompt templates for the workflow's tasks).

3. For each ISSUE or SUGGESTION finding, apply the change by editing the relevant file in place:
   - Edit `${WORKFLOWS_DIR}/workflow.star` for DSL or structural fixes.
   - Edit the appropriate `${WORKFLOWS_DIR}/<task_id>.md` prompt file for prompt-level fixes.
   - Apply all findings; do not skip any ISSUE or SUGGESTION.

4. Write `${CHANGES_PATH}` listing every change made. Format as a markdown list where each entry states:
   - The finding label (ISSUE or SUGGESTION) and a short description of it.
   - The file edited.
   - What was changed.
   If no changes were needed because there were no ISSUE or SUGGESTION findings, write a single line: `No changes required.`

5. Set `outcome` to `applied` if at least one change was made, or `nothing_to_apply` if the review contained no ISSUE or SUGGESTION findings.

Outputs:
- Edit in place: `${WORKFLOWS_DIR}/workflow.star` and any `.md` prompt files in `${WORKFLOWS_DIR}/` that require changes.
- Write: `${CHANGES_PATH}`
- Return JSON with keys:
  - `changes_path`: the absolute path `${CHANGES_PATH}`
  - `outcome`: one of `applied`, `nothing_to_apply`

Do not wrap the JSON in Markdown fences.
