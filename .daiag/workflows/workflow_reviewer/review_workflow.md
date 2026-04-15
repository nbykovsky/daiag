# Review Workflow

Inputs:
- `workflow_id`: ${WORKFLOW_ID}
- `workflows_lib`: ${WORKFLOWS_LIB}
- `report_path`: ${REPORT_PATH}

Instructions:

1. Read `${WORKFLOWS_LIB}/${WORKFLOW_ID}/workflow.star`.
2. Read all `*.md` files in `${WORKFLOWS_LIB}/${WORKFLOW_ID}/` (prompt template files).
3. Review the workflow for:
   - **Structure**: Is the workflow.star well-formed? Are `inputs`, `steps`, `output_artifacts`, and `output_results` all declared? Does `wf = workflow(...)` appear at the top level?
   - **Correctness**: Do artifact paths use `run_dir()`? Are `path_ref` and `json_ref` used correctly for cross-task references? Do each task's `result_keys` match the JSON keys the corresponding prompt template says it will return?
   - **Prompt quality**: Does each prompt template clearly state which files to read and which file to write? Does it end with `Do not wrap the JSON in Markdown fences.`? Are all template placeholders defined in the `vars` map in workflow.star?
   - **Consistency**: Do artifact key names and result key names match between workflow.star and the prompt templates?
4. Write a Markdown review report to `${REPORT_PATH}` (replace if it exists) with these sections:
   - **Summary**: One-paragraph overall assessment of the workflow's quality.
   - **Structure**: Findings about the workflow.star layout and declarations.
   - **Correctness**: Any issues with path refs, json refs, artifact paths, or result keys.
   - **Prompt Quality**: Assessment of each prompt template file by name.
   - **Issues**: Bulleted list of specific problems found. Write "None." if no issues.
   - **Recommendations**: Bulleted list of suggested improvements. Write "None." if no recommendations.

Outputs:
- Write: `${REPORT_PATH}` (replace if it exists)
- Return JSON with keys:
  - `report_path`: absolute path to the written review report file

Do not wrap the JSON in Markdown fences.
