# Task Batch Executor

Execute all pending implementation tasks referenced by the task index.

Inputs:

- task index: "${TASK_INDEX_PATH}"

Outputs:

- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${TASK_INDEX_PATH}" and discover the pending tasks it references.
2. Execute the tasks in order.
3. Stop immediately if any task fails build or test verification.
4. You may update the task index and any referenced task files in place.
5. Write "${STATUS_PATH}" as a short plain-text summary of:
   - final outcome
   - number of tasks completed
   - first blocked or failed task, if any
6. Return JSON only with these keys:
   - `outcome`
   - `task_index_path`
   - `status_path`

Use one of these `outcome` values:

- `complete`
- `failed`
- `blocked`

Do not wrap the JSON in Markdown fences.
