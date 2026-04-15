# Implement Workflow

<!-- DOLLAR{VAR_NAME} is used in this file to describe DOLLAR{VAR_NAME} template placeholders
     in generated prompt files without triggering substitution in this file's own rendering. -->

Inputs:
- `workflows_lib`: ${WORKFLOWS_LIB}
- `blueprint_path`: ${BLUEPRINT_PATH}
- `summary_path`: ${SUMMARY_PATH}
- `WORKFLOWS.md`: ${WORKFLOWS_LIB}/WORKFLOWS.md

Instructions:

1. Read `${BLUEPRINT_PATH}`. If it contains `outcome: needs_clarification`, write `${SUMMARY_PATH}` with the clarification questions from the blueprint. Return `workflow_id` and `workflow_path` as empty strings and `outcome` as `needs_clarification`.

2. Otherwise, implement the workflow described in the blueprint under `${WORKFLOWS_LIB}/<workflow_id>/`:
   - `${WORKFLOWS_LIB}/<workflow_id>/workflow.star`
   - `${WORKFLOWS_LIB}/<workflow_id>/<task_id>.md` — one prompt file per task

### workflow.star rules

```
workflow_id = "<id>"
<var> = input("<name>")
<artifact_path> = format("{run_dir}/<workflow_id>/<artifact>.ext", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["<name>", ...],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "<task_id>",
            prompt = template_file("<task_id>.md", vars = {
                "PLACEHOLDER": <var>,
            }),
            artifacts = {"<key>": artifact(<path>)},
            result_keys = ["<key>", ...],
        ),
    ],
    output_artifacts = {"<key>": path_ref("<task_id>", "<artifact_key>")},
    output_results = {"<key>": json_ref("<task_id>", "<result_key>")},
)
```

Key rules:
- Use `run_dir()` for all run artifact paths: `format("{run_dir}/...", run_dir = run_dir())`.
- Use `projectdir()` only when a prompt needs an absolute path to a project-level source file.
- Use `path_ref("task_id", "artifact_key")` to pass a file from an earlier task to a later one.
- Use `json_ref("task_id", "result_key")` to pass a result value from an earlier task to a later one.
- Every artifact value must use `artifact(...)`.
- `wf = workflow(...)` must be at the top level.
- Declare both `output_artifacts` and `output_results`; use `{}` when a category has no values.
- For iteration: `repeat_until(id=..., max_iters=N, steps=[...], until=eq(json_ref("task_id", "key"), "value"))`.
- For reusing an existing catalog workflow as a stage: use `subworkflow(id="step_id", workflow="catalog_id", inputs={...})` and reference its outputs with `path_ref` and `json_ref`.

### Prompt template rules

Each `<task_id>.md`:

```
# Task Title

Inputs:
- `<name>`: <PLACEHOLDER_VALUE_HERE>

Instructions:
1. ...

Outputs:
- Write/update: <ARTIFACT_PATH_VALUE_HERE>
- Return JSON with keys:
  - `<key>`: <description>

Do not wrap the JSON in Markdown fences.
```

Key rules:
- Use DOLLAR{VAR_NAME} placeholders (matching the `vars` keys in `workflow.star`). Replace the literal word DOLLAR with the dollar sign character when writing prompt files.
- Tell the agent exactly which files to read and write, and the edit semantics (replace / append / update in place).
- List allowed values for enum fields such as `outcome`.
- Every JSON key listed must appear in the task's `result_keys`.
- End with `Do not wrap the JSON in Markdown fences.`

3. Read `${WORKFLOWS_LIB}/WORKFLOWS.md` to understand its current content, then append a new entry:

```
## <workflow_id>

<one-sentence description>

File: `<workflows_lib>/<workflow_id>/workflow.star`

Inputs:
- `<input>` — <description>

Output Artifacts:
- `<key>` — <description>

Output Results: `<key1>`, `<key2>`, ...
```

4. Write a summary to `${SUMMARY_PATH}`:
   - Outcome: `complete`
   - Workflow ID and absolute path to `workflow.star`
   - Tasks created with their prompt files

Outputs:
- Write: `${WORKFLOWS_LIB}/<workflow_id>/workflow.star` and prompt files (path is dynamic; exposed via workflow_path result key)
- Write/update: `${WORKFLOWS_LIB}/WORKFLOWS.md`
- Write: ${SUMMARY_PATH}
- Return JSON with keys:
  - `workflow_id`: the generated workflow ID, or empty string when outcome is needs_clarification
  - `workflow_path`: absolute path to the generated `workflow.star`, or empty string when outcome is needs_clarification
  - `outcome`: one of `complete`, `needs_clarification`

Do not wrap the JSON in Markdown fences.
