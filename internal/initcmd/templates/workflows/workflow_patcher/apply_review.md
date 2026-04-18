# Apply Review

Inputs:
- `report_path`: ${REPORT_PATH}
- `workflow_id`: ${WORKFLOW_ID}
- `workflows_lib`: ${WORKFLOWS_LIB}
- `status_path`: ${STATUS_PATH}

Instructions:

1. Read the review report at `${REPORT_PATH}`.
2. Read `${WORKFLOWS_LIB}/${WORKFLOW_ID}/workflow.star`.
3. Read all `*.md` files in `${WORKFLOWS_LIB}/${WORKFLOW_ID}/` (prompt template files).
4. Examine the **Issues** and **Recommendations** sections of the report.
5. For each actionable issue or recommendation:
   - Edit `${WORKFLOWS_LIB}/${WORKFLOW_ID}/workflow.star` in place if the issue affects workflow structure, artifact declarations, refs, or result_keys.
   - Edit the relevant `${WORKFLOWS_LIB}/${WORKFLOW_ID}/<task_id>.md` prompt template in place if the issue affects prompt clarity, placeholder usage, output instructions, or the `Do not wrap the JSON in Markdown fences.` footer.
   - Do not create new files. Do not delete existing files. Only edit files that already exist under `${WORKFLOWS_LIB}/${WORKFLOW_ID}/`.
6. If the report contained no actionable issues (Issues section says "None." and Recommendations section says "None."), make no edits.
7. Write a brief status summary to `${STATUS_PATH}` listing which files were edited (or "No changes made." if none).

Outputs:
- Edit in place (if needed): `${WORKFLOWS_LIB}/${WORKFLOW_ID}/workflow.star`
- Edit in place (if needed): `${WORKFLOWS_LIB}/${WORKFLOW_ID}/<task_id>.md` files
- Write: `${STATUS_PATH}`
- Return JSON with keys:
  - `outcome`: `complete` if the report contained no actionable issues and no edits were needed, `review_request` if at least one change was applied

Do not wrap the JSON in Markdown fences.
