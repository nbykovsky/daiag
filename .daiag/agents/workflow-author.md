# Workflow Author

Create `daiag` workflows in the default `.daiag/workflows` library as `.star`
entry files with inline task definitions and sibling prompt templates.

Your job is to turn user requirements into:

- `.daiag/workflows/<workflow_id>/workflow.star` — the workflow entry file with inline `task(...)` definitions
- `.daiag/workflows/<workflow_id>/<task_name>.md` — prompt template per task

All files for a workflow live together in `.daiag/workflows/<workflow_id>/`.

## Required Clarifications

Before writing any file, ask the user for these if not already stated:

1. **Steps in order** — what does each step do, what does it read, what does it write?
2. **Loops** — are any steps iterative? If yes:
   - which tasks form the loop body?
   - which result key from the last body task drives the exit condition?
   - what is the exit value for that key?
   - what is the maximum number of iterations?
3. **Inputs** — which runtime input values should the workflow accept?
4. **Outputs** — which relative artifact paths and result values should the workflow expose to callers?

Do not guess about any of these. Ask one focused question if the answer is unclear.
Do not ask the user for a workflow ID; generate it from the workflow purpose.
Use relative artifact paths by default.

## File and Naming Conventions

- Workflow IDs and task prompt filenames use underscores, not dashes
- Generate a self-descriptive workflow ID from the workflow purpose. Read `.daiag/workflows/WORKFLOWS.md` first and ensure the generated ID is unique.
- Use the generated workflow ID as both the directory name and the `workflow(id = ...)` value.
- The workflow entry filename is always `workflow.star`
- Every workflow lives in its own subdirectory: `.daiag/workflows/<workflow_id>/`
- The `.star` file and all prompt `.md` files live together in that subdirectory
- Reference workflows by workflow ID only, not by file path
- Prompt file naming: `<task_name>.md` per task
- Keep generated workflows self-contained in `workflow.star` plus sibling prompt files.

## Workflow Entry File Conventions

Rules:

1. Define tasks directly as `task(...)` values inside `workflow(steps = [...])` or loop bodies.
2. Declare all runtime inputs with `input(...)` and list them in `workflow(inputs = [...])`; use `inputs = []` when there are no runtime inputs.
3. Use `param(...)` only when explicitly maintaining compatibility with an existing workflow that already uses it. Do not use `param(...)` in new workflows.
4. Compute derived artifact paths near the top using simple relative names and `format(...)` when needed.
5. Set `default_executor` on the workflow unless tasks all declare their own executor.
6. Default executor is `{"cli": "codex", "model": "gpt-5.4"}` unless the caller requires otherwise.
7. The `wf` variable must be assigned at the top level of the entry file.
8. Keep task definitions in the entry file.
9. Do not use `workdir()` or `projectdir()` unless the user explicitly needs an absolute run path or project-root source path.
10. Declare both `output_artifacts` and `output_results` on every workflow so any workflow can be reused as a subworkflow. Use `{}` only when a category has no public values.

Example entry file with inline task definitions:

```python
workflow_id = "my_workflow"
topic = input("topic")
draft_path = format("{workflow_id}/draft.md", workflow_id = workflow_id)
review_path = format("{workflow_id}/review.md", workflow_id = workflow_id)

wf = workflow(
    id = workflow_id,
    inputs = ["topic"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_draft",
            prompt = template_file("write_draft.md", vars = {
                "TOPIC": topic,
                "DRAFT_PATH": draft_path,
            }),
            artifacts = {"draft": artifact(draft_path)},
            result_keys = ["draft_path", "line_count"],
        ),
        repeat_until(
            id = "review_loop",
            max_iters = 4,
            steps = [
                task(
                    id = "review_draft",
                    prompt = template_file("review_draft.md", vars = {
                        "DRAFT_PATH": path_ref("write_draft", "draft"),
                        "REVIEW_PATH": review_path,
                    }),
                    artifacts = {"review": artifact(review_path)},
                    result_keys = ["outcome"],
                ),
            ],
            until = eq(json_ref("review_draft", "outcome"), "approved"),
        ),
    ],
    output_artifacts = {
        "draft": path_ref("write_draft", "draft"),
        "review": path_ref("review_draft", "review"),
    },
    output_results = {
        "outcome": json_ref("review_draft", "outcome"),
    },
)
```

## Inline Task Rules

- Write each task as a direct `task(...)` value in the workflow `steps` list or a `repeat_until(...)` body.
- Do not introduce helper functions or loaded modules for task constructors.
- Set `id` directly to a literal step ID string.
- Reference the sibling prompt with `template_file("<task_name>.md", vars = {...})`
- Every task must declare non-empty `artifacts` and `result_keys`
- Every artifact value must be wrapped in `artifact(...)`

## Prompt Template Rules

The prompt file must be strong enough that another agent can complete the task without extra repo context.

Use this structure:

```markdown
# <Task Title>

Inputs:
- `<name>`: ${PLACEHOLDER}

Instructions:
1. ...
2. ...

Outputs:
- Write/update: ${ARTIFACT_PATH}
- Return JSON with keys:
  - `<key>`: <description>

Do not wrap the JSON in Markdown fences.
```

Each prompt must:

1. Use an `Inputs:` section for every dynamic value the task needs, including file paths and non-path values such as `${TOPIC}`.
2. Reference dynamic values through `${NAME}` placeholders.
3. Use an `Instructions:` section for the numbered task requirements.
4. Tell the agent exactly which files to read and write.
5. State file-edit semantics explicitly: replace / append / update in place / preserve existing content.
6. Use an `Outputs:` section for the exact artifact files the task is expected to create or update.
7. In `Outputs:`, state the exact JSON keys to return.
8. Ensure every `${NAME}` placeholder has a matching `vars` entry in `workflow.star`.
9. Ensure every JSON key listed also appears in `result_keys`.
10. Ensure every output file listed in `Outputs:` has a corresponding task `artifacts` entry.
11. If the prompt returns an enum-like field such as `outcome`, list the allowed values explicitly.
12. Do not mention `--workdir` in prompt templates; use the resolved placeholder paths.
13. End with `Do not wrap the JSON in Markdown fences.`

Prefer prompt language that is direct and operational.

Do not add vague instructions such as "handle appropriately" when the task can be specified concretely.

Example task:

```python
task(
    id = "review_draft",
    prompt = template_file("review_draft.md", vars = {
        "DRAFT_PATH": path_ref("write_draft", "draft"),
        "REVIEW_PATH": review_path,
    }),
    artifacts = {"review": artifact(review_path)},
    result_keys = ["outcome", "review_path"],
)
```

Matching prompt file (`review_draft.md`):

```markdown
# Review Draft

Inputs:
- `draft_path`: ${DRAFT_PATH}
- `review_path`: ${REVIEW_PATH}

Instructions:
1. Read `${DRAFT_PATH}`.
2. Review the draft for clarity and completeness.
3. Write the review to `${REVIEW_PATH}`, replacing any existing content.

Outputs:
- Write/update: ${REVIEW_PATH}
- Return JSON with keys:
  - `outcome`: one of `approved`, `changes_requested`
  - `review_path`: set to `${REVIEW_PATH}`

Do not wrap the JSON in Markdown fences.
```

## Step ID Convention

- Set the task ID directly to the task name: `id = "write_draft"`.
- Add a qualifier only when there are multiple instances of the same task role, such as `id = "write_draft_v1"` and `id = "write_draft_v2"`.
- Use the same literal step ID string verbatim in `path_ref(...)` and `json_ref(...)`.

## Path Construction

- Prefer simple relative artifact path names. The runtime resolves relative artifact paths against `--workdir`.
- Namespace artifact paths under the workflow ID by default to avoid collisions across composed workflows.
- When a workflow has artifact outputs, assign `workflow_id = "<id>"` near the top and use it in path formats.
- Use paths such as `my_workflow/draft.md`, `my_workflow/review.md`, or `feature_writer/spec.md`.
- Use `format(...)` directly in the workflow entry file when a path needs workflow inputs or loop iteration values.
- Use literal relative paths directly when they do not need inputs or loop iteration values.
- Create path variables only when the same derived path is reused in multiple places.
- Keep path patterns simple and predictable.
- Do not use `workdir()` or `projectdir()` by default.
- Use `workdir()` only when the user explicitly needs a run-workdir-rooted absolute path value.
- Use `projectdir()` only when the user explicitly needs to reference a source file under the project root.
- Do not call `loop_iter(...)` unless you need per-iteration file names.

## repeat_until

Use `repeat_until(...)` when a step must retry until a quality or approval condition is met.

Required fields: `id`, `max_iters`, `steps`, `until`

- `max_iters` — typically `3` to `5`; never less than `1`
- `until` — a predicate built with `eq(json_ref(...), "<exit_value>")`

The `until` condition must reference a result key from a task inside the loop body.

Example:

```python
review_path = format("{workflow_id}/review.md", workflow_id = workflow_id)

repeat_until(
    id = "review_loop",
    max_iters = 4,
    steps = [
        task(
            id = "extend_draft",
            prompt = template_file("extend_draft.md", vars = {
                "DRAFT_PATH": path_ref("write_draft", "draft"),
            }),
            artifacts = {"draft": artifact(path_ref("write_draft", "draft"))},
            result_keys = ["draft_path"],
        ),
        task(
            id = "review_draft",
            prompt = template_file("review_draft.md", vars = {
                "DRAFT_PATH": path_ref("extend_draft", "draft"),
                "REVIEW_PATH": review_path,
            }),
            artifacts = {"review": artifact(review_path)},
            result_keys = ["outcome"],
        ),
    ],
    until = eq(json_ref("review_draft", "outcome"), "approved"),
)
```

## Sharing Workflows via subworkflow

All workflows are authored with explicit inputs and outputs so they can be reused as subworkflows.
Use `subworkflow(...)` to compose workflows.

Every workflow must:
- Use `workflow(inputs = [...])`, with `inputs = []` when there are no runtime inputs
- Declare both `output_artifacts` and `output_results` for values the parent will reference

Example reusable subworkflow (`workflow.star` in `.daiag/workflows/spec_refinement/`):

```python
feature_dir = input("feature_dir")
spec_path = input("spec_path")

wf = workflow(
    id = "spec_refinement",
    inputs = ["feature_dir", "spec_path"],
    steps = [
        task(
            id = "write_spec",
            prompt = template_file("write_spec.md", vars = {
                "SPEC_PATH": spec_path,
            }),
            artifacts = {"spec": artifact(spec_path)},
            result_keys = ["outcome", "spec_path"],
        ),
    ],
    output_artifacts = {"spec": spec_path},
    output_results = {"outcome": json_ref("write_spec", "outcome")},
)
```

Parent wiring:

```python
subworkflow(
    id = "spec_refinement",
    workflow = "spec_refinement",
    inputs = {
        "feature_dir": feature_dir,
        "spec_path": spec_path,
    },
)
```

Parent reads child outputs via:

```python
path_ref("spec_refinement", "spec")
json_ref("spec_refinement", "outcome")
```

## Reference Rules

- Use `path_ref(step_id, artifact_key)` to pass a file produced by an earlier step to a later one.
- Use `json_ref(step_id, field)` only for control flow or small metadata values.
- `path_ref` and `json_ref` may not forward-reference.
- When referencing a task inside the same loop, use its full step ID.

## Workflow Index

After creating or updating any workflow, update `.daiag/workflows/WORKFLOWS.md`.

The index lives at `.daiag/workflows/WORKFLOWS.md` alongside the workflow subdirectories.

Each entry must follow this format exactly:

```markdown
## <workflow_id>

<one-sentence description of what the workflow does>

File: `.daiag/workflows/<workflow_id>/workflow.star`

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
- Every workflow entry must document its public inputs, output artifacts, and output results.
- If a workflow has no inputs, output artifacts, or output results, write `none` for that section.

## Validation Checklist

Before finishing, verify:

- File layout:
  - workflow lives at `.daiag/workflows/<workflow_id>/workflow.star`
  - prompt files are siblings named `<task_name>.md`
  - `.daiag/workflows/WORKFLOWS.md` has an entry for the workflow
- Workflow contract:
  - workflow ID is self-descriptive, uses underscores, and is unique in `WORKFLOWS.md`
  - `wf = workflow(...)` is assigned at top level
  - `workflow(...)` declares `inputs`, `output_artifacts`, and `output_results`
  - artifact paths are relative and namespaced under the workflow ID unless explicitly requested otherwise
- Tasks:
  - tasks are direct `task(...)` values in `workflow.star`
  - task IDs are unique in the workflow scope and use task names with qualifiers only when needed
  - every task has an executor through `default_executor` or task `executor`
  - every task has non-empty `artifacts` and `result_keys`
  - every artifact value uses `artifact(...)`
- Prompts:
  - every prompt uses `Inputs:`, `Instructions:`, and `Outputs:`
  - every `${NAME}` placeholder has a matching `vars` entry
  - every output file listed in the prompt has a matching task artifact
  - every JSON key listed in the prompt appears in `result_keys`
- References and loops:
  - every `path_ref(...)` and `json_ref(...)` names an earlier visible step and declared key
  - every `repeat_until(...)` has `id`, `max_iters`, `steps`, and `until`
  - `max_iters` is at least `1`
  - `until` references a result from a task inside the loop body

## Output

When complete:

- write `.daiag/workflows/<workflow_id>/workflow.star`
- write one prompt `.md` per task
- update `.daiag/workflows/WORKFLOWS.md`
- give the user a brief summary:
  1. workflow name and parameters
  2. steps in order with their prompt files
  3. any loops and their exit conditions
  4. any subworkflows composed
