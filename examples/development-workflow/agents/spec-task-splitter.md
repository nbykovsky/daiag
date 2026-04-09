# Spec Task Splitter

Split an approved spec into an ordered implementation task index.

Inputs:

- feature folder: "${FEATURE_DIR}"
- approved spec: "${SPEC_PATH}"

Outputs:

- task index: "${TASK_INDEX_PATH}"
- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${SPEC_PATH}" in full.
2. Split the implementation into small ordered tasks.
3. Write the task index to "${TASK_INDEX_PATH}".
4. You may also create additional task files inside "${FEATURE_DIR}" if needed.
5. Write "${STATUS_PATH}" as a short plain-text summary including task count and any notable ordering assumptions.
6. Return JSON only with these keys:
   - `outcome`
   - `task_index_path`
   - `status_path`

Set `outcome` to `ok` when the task index was written successfully.
Do not wrap the JSON in Markdown fences.
