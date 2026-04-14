# Implement Workflow from Description Document

Inputs:
- `description_path`: ${BLUEPRINT_PATH}
- `workflows_lib`: ${WORKFLOWS_LIB}
- `summary_path`: ${SUMMARY_PATH}

Instructions:
1. Read the workflow description document at `${BLUEPRINT_PATH}`. Treat it as the user's requirements document. It may be a free-form description or a structured natural-language blueprint.
2. Read `${WORKFLOWS_LIB}/WORKFLOWS.md`. Use it as the catalog of existing reusable workflows and their public input/output contracts.
3. Treat `${WORKFLOWS_LIB}` as the target workflow catalog. Before writing, confirm that `${WORKFLOWS_LIB}/WORKFLOWS.md` exists. If it does not exist, do not create a new workflow catalog; return `needs_clarification` and explain that the selected workflow catalog is missing.
4. Before writing files, derive an implementation contract from the document. The contract must identify:
   - the main workflow purpose and generated workflow ID
   - runtime inputs accepted by the main workflow
   - final output artifacts and result values exposed by the main workflow
   - stages in execution order
   - whether each stage is implemented as a new task, a new child workflow, or an existing reusable workflow
   - for every existing reusable workflow stage: workflow ID, required input bindings, and public outputs used later
   - for every new task or child workflow: what it reads, what it writes, and which JSON result keys it returns
   - any loops: loop body stages, result key that drives exit, exit value, and maximum iteration count
5. If any required information cannot be inferred safely, write `${SUMMARY_PATH}` with a `Needs clarification` section and return `outcome` as `needs_clarification`. Do this before creating or modifying workflow files. Required information includes:
   - missing or ambiguous stage order
   - missing or ambiguous runtime inputs
   - missing or ambiguous final outputs
   - a stage whose read/write behavior is unclear
   - a file artifact whose path or file type is unclear
   - an enum-like result value without allowed values
   - an iterative stage without a clear loop body, exit result key, exit value, or maximum iteration count
   - a handoff that cannot be mapped to a prior stage's public artifact or result
   - a requested existing workflow whose catalog contract does not provide the required input or output
   - a workflow ID collision where the existing catalog entry does not clearly match the requested behavior
6. If clarification is needed, do not write any workflow files. Return JSON with:
   - `workflow_id`: empty string
   - `workflow_path`: empty string
   - `outcome`: `needs_clarification`

## Inference Rules

- If the document describes one standalone operation, create one workflow and assume no loop unless the document says otherwise.
- If the document mentions `given <value>` or lists a starting input, treat it as a runtime input.
- If the document asks to create, write, or save an artifact but does not specify a path, use `<workflow_id>/<artifact_name>.<ext>`.
- If an output result key is obvious, use it without asking; for example, a poem artifact should return `poem_path`.
- Do not ask for a workflow ID. Generate a short, self-descriptive ID from the workflow purpose using lowercase words joined with underscores.
- If a generated ID already exists and the document does not explicitly request modifying that workflow, generate a unique variant such as `<id>_v2` or a more specific ID. Do not skip creation silently.
- If the document explicitly asks to modify an existing workflow, update that workflow and its `WORKFLOWS.md` entry instead of generating a new ID.
- Use relative artifact paths by default.

## Existing Workflow Reuse

- Prefer existing workflows from `${WORKFLOWS_LIB}/WORKFLOWS.md` when their public contract satisfies a stage.
- Reference existing workflows by workflow ID only, not by file path.
- Bind every required input of an existing workflow.
- Do not bind unknown inputs.
- Use only declared output artifacts and output results from the existing workflow's catalog entry.
- Never reference internal task IDs from a child workflow.
- If the catalog entry is missing a required input/output detail, return `needs_clarification` instead of guessing from implementation files.

## Workflow File Rules

For every workflow you create or update:

- Place files in `${WORKFLOWS_LIB}/<workflow_id>/`.
- Write the entry file as `${WORKFLOWS_LIB}/<workflow_id>/workflow.star`.
- Use the same workflow ID for the directory name and the workflow declaration.
- Assign `wf = workflow(...)` at the top level.
- Declare all runtime inputs and use `inputs = []` when there are no runtime inputs.
- Use `param(...)` only when explicitly maintaining compatibility with an existing workflow that already uses it. Do not use `param(...)` in new workflows.
- Compute derived artifact paths near the top using simple relative names and `format(...)` when needed.
- Use `default_executor = {"cli": "codex", "model": "gpt-5.4"}` unless the document requires a different executor or all tasks declare their own executor.
- Define tasks directly as `task(...)` values inside `workflow(steps = [...])` or loop bodies.
- Do not introduce helper functions or loaded modules for task constructors.
- Declare both `output_artifacts` and `output_results`; use `{}` when a category has no public values.
- Keep generated workflows self-contained in `workflow.star` plus sibling prompt files.

## Task Rules

- Set each task ID directly to the task name, using lowercase words joined with underscores.
- Add a qualifier only when there are multiple instances of the same task role.
- Task IDs must be unique in the workflow scope.
- Reference sibling prompts with `template_file("<task_name>.md", vars = {...})`.
- Every task must declare non-empty `artifacts` and `result_keys`.
- Every artifact value must be wrapped in `artifact(...)`.
- Use `path_ref(step_id, artifact_key)` to pass a file produced by an earlier step to a later step.
- Use `json_ref(step_id, field)` only for control flow or small metadata values.
- Do not forward-reference steps.

## Prompt Template Rules

Write one prompt file `${WORKFLOWS_LIB}/<workflow_id>/<task_name>.md` per task.

Each prompt must:

- start with `# <Task Title>`
- include an `Inputs:` section for every dynamic value the task needs, including file paths and non-path values
- reference dynamic values with dollar-brace placeholders, such as a placeholder named `NAME`; write these in generated prompt files as a dollar sign followed immediately by an opening brace, the placeholder name, and a closing brace
- include an `Instructions:` section with numbered, concrete task requirements
- tell the executing agent exactly which files to read and write
- state file-edit semantics explicitly: replace, append, update in place, or preserve existing content
- include an `Outputs:` section for the exact artifact files the task is expected to create or update
- state the exact JSON keys to return
- ensure every dollar-brace placeholder has a matching `vars` entry in `workflow.star`
- ensure every JSON key listed in the prompt appears in the task's `result_keys`
- ensure every output file listed in the prompt has a corresponding task artifact
- list allowed values for enum-like fields such as `outcome`
- end with `Do not wrap the JSON in Markdown fences.`

Do not use vague instructions such as "handle appropriately" when the behavior can be specified concretely.

Use this prompt shape:

```markdown
# <Task Title>

Inputs:
- `<name>`: <placeholder for the matching dynamic value>

Instructions:
1. ...
2. ...

Outputs:
- Write/update: <artifact path placeholder>
- Return JSON with keys:
  - `<key>`: <description>

Do not wrap the JSON in Markdown fences.
```

## Loop Rules

Use a loop only when the document requires iterative behavior.

For every loop:

- set a unique loop ID
- set `max_iters` to a value from the document, or return `needs_clarification` if the maximum cannot be inferred
- define the loop body stages in order
- set the exit condition from a result key returned by a task inside the loop body
- use an equality exit value when the document gives one
- ensure `max_iters` is at least `1`

Do not invent quality-review or retry loops that the document did not request.

## WORKFLOWS.md Rules

After creating or updating any workflow, update `${WORKFLOWS_LIB}/WORKFLOWS.md`.

Each entry must follow this format:

```markdown
## <workflow_id>

<one-sentence description of what the workflow does>

File: `<workflow_path>`

Inputs:
- `<input>` — <description>

Output Artifacts:
- `<key>` — <path or description>

Output Results: `<key1>`, `<key2>`, ...
```

Rules:

- Add a new `##` section for each new workflow.
- Update the existing section when a workflow's inputs, outputs, or description change.
- Keep entries in the order they were added.
- Every workflow entry must document public inputs, output artifacts, and output results.
- If a workflow has no inputs, output artifacts, or output results, write `none` for that section.

## Self-Check Before Completion

Before returning `complete`, verify:

- every created or updated workflow has a `WORKFLOWS.md` entry
- every workflow ID is self-descriptive, uses underscores, and is unique unless intentionally updating an existing workflow
- every `workflow.star` assigns `wf = workflow(...)` at top level
- every workflow declares inputs, output artifacts, and output results
- artifact paths are relative and namespaced under the workflow ID unless the document explicitly requested an existing file path
- every task has an executor through `default_executor` or task `executor`
- every task has non-empty artifacts and result keys
- every artifact uses `artifact(...)`
- every prompt uses `Inputs:`, `Instructions:`, and `Outputs:`
- every prompt placeholder has a matching `vars` entry
- every output file in a prompt has a matching task artifact
- every JSON key in a prompt appears in `result_keys`
- every reference names an earlier visible step and declared key
- every loop has `id`, `max_iters`, body steps, and an exit condition that references a result from inside the loop body

## Summary

Write `${SUMMARY_PATH}` with:

- `Outcome`: `complete` or `needs_clarification`
- `Main workflow`: workflow ID and absolute path, or `none` when clarification is needed
- `Created workflows`: workflow IDs and paths
- `Updated workflows`: workflow IDs and paths
- `Prompt files`: prompt file paths
- `Subworkflows composed`: workflow IDs reused or created as child stages
- `Loops`: loop IDs and exit conditions, or `none`
- `Clarification questions`: questions when clarification is needed, or `none`

Outputs:
- Write/update: ${SUMMARY_PATH}
- Return JSON with keys:
  - `workflow_id`: generated workflow ID, or empty string when `outcome` is `needs_clarification`
  - `workflow_path`: absolute path to the main workflow `.star` file, or empty string when `outcome` is `needs_clarification`
  - `outcome`: one of `complete`, `needs_clarification`

Do not wrap the JSON in Markdown fences.
