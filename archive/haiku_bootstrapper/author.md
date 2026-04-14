You are a workflow author. You will implement a daiag workflow based on a blueprint.

## Blueprint

Read the blueprint file at: ${BLUEPRINT_PATH}

## Workflow Library

The workflow catalog directory is: ${WORKFLOWS_LIB}

## Your task

Implement the workflow described in the blueprint. Create these files:

1. `${WORKFLOWS_LIB}/<workflow_id>/workflow.star` — the Starlark workflow definition
2. `${WORKFLOWS_LIB}/<workflow_id>/<step_id>.md` — one prompt template per step
3. Write a summary to: ${SUMMARY_PATH}

### workflow.star rules

- Top-level: declare inputs, compute paths, define `wf = workflow(...)`
- Use `input("name")` for each workflow input
- Use `run_dir()` for run artifact paths: `format("{run_dir}/step_id/output.md", run_dir = run_dir())`
- Use `projectdir()` if you need the project root
- Each task needs: `id`, `prompt = template_file("name.md", vars={...})`, `artifacts = {"key": artifact(path)}`, `result_keys = ["key1", ...]`
- Declare `output_artifacts` and `output_results` on the workflow using `path_ref(...)` and `json_ref(...)`
- Executor: `{"cli": "claude", "model": "claude-haiku-4-5-20251001"}`

### Prompt template rules (the .md files)

- Use `$\{VAR_NAME\}` placeholders for variables passed in `vars={}`
- The prompt must tell the executor exactly what file to write and where
- The prompt must end with "Return ONLY a JSON object: {<result_keys>}"

### Summary file

Write a brief Markdown summary to ${SUMMARY_PATH} describing what was created.

Make sure all parent directories exist before writing files.

## Important constraints

- `workflow_id` must match `[a-z0-9_-]+` and must match the directory name you created
- `workflow_path` must be the **absolute path** to the workflow.star file you created
- The workflow.star `id` field must equal the `workflow_id`

After creating all files, return ONLY this JSON (no other text before or after):
{"workflow_id": "<the_workflow_id>", "workflow_path": "<absolute_path_to_workflow.star>", "outcome": "complete"}
