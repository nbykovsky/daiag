# Implement Workflow

<!-- DOLLAR{VAR_NAME} is used in this file to describe DOLLAR{VAR_NAME} template placeholders
     in generated prompt files without triggering substitution in this file's own rendering. -->

Inputs:
- `workflows_lib`: ${WORKFLOWS_LIB}
- `composition_plan_path`: ${COMPOSITION_PLAN_PATH}
- `summary_path`: ${SUMMARY_PATH}

Instructions:

1. Read the composition plan at `${COMPOSITION_PLAN_PATH}`. Extract:
   - The workflow ID and inputs
   - The full step structure: step types (task, subworkflow, repeat_until, when), their order, and nesting
   - Data flow: which `path_ref` and `json_ref` connections pass data between steps
   - Output artifacts and output results

2. Read `${WORKFLOWS_LIB}/WORKFLOWS.md` to verify the exact input/output contracts of any catalog workflows used as subworkflow steps.

3. Implement the workflow under `${WORKFLOWS_LIB}/<workflow_id>/`:
   - `${WORKFLOWS_LIB}/<workflow_id>/workflow.star`
   - `${WORKFLOWS_LIB}/<workflow_id>/<task_id>.md` — one prompt file per direct `task` step

### workflow.star rules

Top-level declarations:
```
workflow_id = "<id>"
<input_var> = input("<name>")
<artifact_path> = format("{run_dir}/<workflow_id>/<name>.md", run_dir = run_dir())
```

For artifact paths scoped to a loop iteration, declare at the top level:
```
<loop_path> = format("{run_dir}/<workflow_id>/<name>_{iter}.md",
    run_dir = run_dir(),
    iter = loop_iter(loop_id = "<loop_id>"))
```

Workflow structure:
```
wf = workflow(
    id = workflow_id,
    inputs = ["<name>", ...],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [...],
    output_artifacts = {"<key>": path_ref("<step_id>", "<artifact_key>"), ...},
    output_results = {"<key>": json_ref("<step_id>", "<result_key>"), ...},
)
```

Step type reference:

**task** — direct executor step:
```
task(
    id = "<task_id>",
    prompt = template_file("<task_id>.md", vars = {"<PLACEHOLDER>": <value_or_ref>}),
    artifacts = {"<key>": artifact(<path>)},
    result_keys = ["<key>", ...],
)
```

**subworkflow** — delegate to a catalog workflow:
```
subworkflow(
    id = "<step_id>",
    workflow = "<catalog_workflow_id>",
    inputs = {"<key>": <value_or_ref>},
)
```

**repeat_until** — loop until a condition is met:
```
repeat_until(
    id = "<loop_id>",
    max_iters = N,
    steps = [<child steps>],
    until = eq(json_ref("<step_id>", "<key>"), "<value>"),
)
```

**when** — conditional branch:
```
when(
    id = "<when_id>",
    condition = eq(json_ref("<step_id>", "<key>"), "<value>"),
    steps = [<true branch steps>],
    else_steps = [],
)
```

Reference rules:
- `path_ref("<step_id>", "<artifact_key>")` — artifact path from an earlier step
- `json_ref("<step_id>", "<result_key>")` — result value from an earlier step
- Steps inside `when(...)` are NOT visible to later parent steps or workflow outputs
- Steps inside `repeat_until(...)` ARE visible to later steps and workflow outputs
- `artifact(...)` must be non-empty; every value must be wrapped with `artifact(...)`

### Prompt template rules

Each `<task_id>.md` prompt file:
- List all inputs using DOLLAR{PLACEHOLDER} substitution (replace the word DOLLAR with $ when writing the file)
- Tell the agent exactly which files to read and which file to write
- List allowed values for enum result fields such as `outcome`
- Every JSON key listed must appear in the task's `result_keys`
- End with: `Do not wrap the JSON in Markdown fences.`

4. Read `${WORKFLOWS_LIB}/WORKFLOWS.md`, then append a new entry:

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

5. Write a summary to `${SUMMARY_PATH}`:
   - Outcome: `complete`
   - Workflow ID and absolute path to `workflow.star`
   - Each step with its type (task / subworkflow / repeat_until / when)

Outputs:
- Write: `${WORKFLOWS_LIB}/<workflow_id>/workflow.star` and one `<task_id>.md` per direct task
- Write/update: `${WORKFLOWS_LIB}/WORKFLOWS.md`
- Write: `${SUMMARY_PATH}`
- Return JSON with keys:
  - `workflow_id`: the generated workflow ID
  - `workflow_path`: absolute path to the generated `workflow.star`
  - `outcome`: `complete`

Do not wrap the JSON in Markdown fences.
