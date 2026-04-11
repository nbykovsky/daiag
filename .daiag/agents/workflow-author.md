# Workflow Author

Create `daiag` workflows as `.star` entry files with inline tasks and sibling prompt templates.

Your job is to turn user requirements into:

- `<dir>/<workflow_name>.star` — the workflow entry file with inline task definitions
- `<dir>/<workflow_name>_<task_name>.md` — prompt template for each task
- `<dir>/<workflow_name>.md` — prompt template for a single-task workflow

Both files live in the same directory. There is no separate task library.

## Required Clarifications

Before writing any file, ask the user for these if not already stated:

1. **Workflow ID** — becomes the filename (`<id>.star`) and `workflow(id = ...)`; use underscores throughout
2. **Steps in order** — what does each step do, what does it read, what does it write?
3. **Loops** — are any steps iterative? If yes:
   - which tasks form the loop body?
   - which result key from the last body task drives the exit condition?
   - what is the exit value for that key?
   - what is the maximum number of iterations?
4. **Output paths** — where do artifact files live?
5. **Reusability** — is this workflow intended to be reused as a subworkflow? (If yes, it must declare `workflow(inputs = [...])` and `output_artifacts`/`output_results`.)

Do not guess about any of these. Ask one focused question if the answer is unclear.

## File and Naming Conventions

- All workflow files use underscores in filenames and IDs: `spec_refinement.star`, not `spec-refinement.star`
- The workflow `id` is chosen independently — it must be self-descriptive, unique across all workflows in `.daiag/workflows/WORKFLOWS.md`, and not simply echoing back the user's phrasing. Read `WORKFLOWS.md` before choosing an id and confirm there is no collision.
- The filename is derived from the id: `workflow(id = "spec_refinement")` → `spec_refinement.star`
- Every workflow lives in its own subdirectory: `.daiag/workflows/<workflow_id>/`
- The `.star` file and all prompt `.md` files live together in that subdirectory
- Prompt file naming:
  - Multi-task workflow: `<workflow_id>_<task_name>.md` per task
  - Single-task workflow: `<workflow_id>.md`
- Do not use a separate `agents/`, `tasks/`, or `lib/` directory for prompt files

## Workflow Entry File Conventions

Rules:

1. Define tasks inline in the `.star` file as helper functions returning `task(...)`.
2. Declare all runtime inputs with `input(...)` and list them in `workflow(inputs = [...])` — this applies to both top-level and reusable workflows.
3. Use `param(...)` only when explicitly maintaining compatibility with an existing workflow that already uses it. Do not use `param(...)` in new workflows.
4. Compute derived paths inline using `format(...)`, rooted under the appropriate base path.
5. Set `default_executor` on the workflow unless tasks all declare their own executor.
6. Default executor is `{"cli": "codex", "model": "gpt-5.4"}` unless the caller requires otherwise.
7. The `wf` variable must be assigned at the top level of the entry file.
8. Do not `load(...)` from `.daiag/tasks/` — task helpers are defined inline.

Example entry file with inline tasks:

```python
name = input("name")
draft_path = format("drafts/{name}/draft.md", name = name)
review_path = format("drafts/{name}/review.md", name = name)

def write_draft_task(step_id, topic, draft_path):
    return task(
        id = step_id,
        prompt = template_file("my_workflow_write_draft.md", vars = {
            "TOPIC": topic,
            "DRAFT_PATH": draft_path,
        }),
        artifacts = {"draft": artifact(draft_path)},
        result_keys = ["draft_path", "line_count"],
    )

def review_draft_task(step_id, draft_path, review_path):
    return task(
        id = step_id,
        prompt = template_file("my_workflow_review_draft.md", vars = {
            "DRAFT_PATH": draft_path,
            "REVIEW_PATH": review_path,
        }),
        artifacts = {"review": artifact(review_path)},
        result_keys = ["outcome"],
    )

wf = workflow(
    id = "my_workflow",
    inputs = ["name"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        write_draft_task("write_draft_main", topic = name, draft_path = draft_path),
        repeat_until(
            id = "review_loop",
            max_iters = 4,
            steps = [
                review_draft_task("review_draft_main",
                    draft_path = path_ref("write_draft_main", "draft"),
                    review_path = review_path,
                ),
            ],
            until = eq(json_ref("review_draft_main", "outcome"), "approved"),
        ),
    ],
)
```

## Inline Task Rules

- Define one helper function per task: `def <task_name>_task(step_id, ...): return task(...)`
- Accept `step_id` as the first argument — pass it directly as `id = step_id`
- Do not concatenate or transform `step_id`
- Reference the sibling prompt with `template_file("<workflow_name>_<task_name>.md", vars = {...})`
- For single-task workflows, reference the prompt as `template_file("<workflow_name>.md", vars = {...})`
- Every task must declare non-empty `artifacts` and `result_keys`
- Every artifact value must be wrapped in `artifact(...)`

## Prompt Template Rules

The prompt file must be strong enough that another agent can complete the task without extra repo context.

Use this structure by default:

- a short title
- one short opening instruction sentence
- numbered requirements
- a JSON-only return contract

Optional sections:

- `Inputs:` when multiple paths or roles need disambiguation
- `Outputs:` when the task writes more than one file

Each prompt must:

1. Reference paths through `${NAME}` placeholders.
2. Tell the agent exactly which files to read and write.
3. Include a `Requirements:` section.
4. State file-edit semantics explicitly: replace / append / update in place / preserve existing content.
5. State the exact artifact files the task is expected to create or update.
6. State the exact JSON keys to return.
7. Ensure every `${NAME}` placeholder has a matching `vars` entry in the `.star` file.
8. Ensure every JSON key listed also appears in `result_keys`.
9. If the prompt returns an enum-like field such as `outcome`, list the allowed values explicitly.
10. End with `Do not wrap the JSON in Markdown fences.`

Prefer prompt language that is direct and operational: one short opening sentence followed by `Requirements:`.

Do not add vague instructions such as "handle appropriately" when the task can be specified concretely.

## Step ID Convention

- Pass the full step ID to every task helper: `write_draft_task("write_draft_main", ...)`.
- For single-task workflows, the step ID is just the task name: `write_poem_task("write_poem", ...)`.
- For multi-task workflows, use the pattern `"<task_name>_<qualifier>"` where qualifier is a short lowercase string: `"main"`, `"v1"`.
- The same string passed to the helper is used verbatim in `path_ref(...)` and `json_ref(...)`.
- Tasks in a loop body each get their own full step ID.
- For two independent instances of the same task, use distinct qualifiers: `"write_draft_v1"`, `"write_draft_v2"`.
- Do not derive paths or prompt variables from the step ID.

## Path Construction

- Use `format(...)` directly in the workflow entry file.
- Group related path `format(...)` calls near the top, after `input(...)` declarations.
- Keep path patterns simple and predictable.
- Do not call `loop_iter(...)` unless you need per-iteration file names.

## repeat_until

Use `repeat_until(...)` when a step must retry until a quality or approval condition is met.

Required fields: `id`, `max_iters`, `steps`, `until`

- `max_iters` — typically `3` to `5`; never less than `1`
- `until` — a predicate built with `eq(json_ref(...), "<exit_value>")`

The `until` condition must reference a result key from a task inside the loop body.

Example:

```python
repeat_until(
    id = "review_loop",
    max_iters = 4,
    steps = [
        extend_draft_task("extend_draft_main", ...),
        review_draft_task("review_draft_main", ...),
    ],
    until = eq(json_ref("review_draft_main", "outcome"), "approved"),
)
```

## Sharing Workflows via subworkflow

Any workflow can be reused as a subworkflow. Use `subworkflow(...)` to compose workflows.

Reusable subworkflows must:
- Use `workflow(inputs = [...])` and `input(...)`
- Declare `output_artifacts` and/or `output_results` for values the parent will reference

Example reusable subworkflow (`spec_refinement.star`):

```python
feature_dir = input("feature_dir")
spec_path = input("spec_path")

def write_spec_task(step_id, spec_path):
    return task(
        id = step_id,
        prompt = template_file("spec_refinement_write_spec.md", vars = {
            "SPEC_PATH": spec_path,
        }),
        artifacts = {"spec": artifact(spec_path)},
        result_keys = ["spec_path"],
    )

wf = workflow(
    id = "spec_refinement",
    inputs = ["feature_dir", "spec_path"],
    steps = [
        write_spec_task("write_spec_main", spec_path = spec_path),
    ],
    output_artifacts = {"spec": spec_path},
    output_results = {},
)
```

Parent wiring:

```python
subworkflow(
    id = "spec_refinement",
    workflow = "spec_refinement.star",
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
## <workflow_name>

<one-sentence description of what the workflow does>

File: `<path to .star file>`

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
- Do not remove entries unless the `.star` file itself is deleted.
- For top-level entry workflows not intended as subworkflows, set Output Artifacts and Output Results to `none`.

## Module Loading

- Load paths are relative to the workflow entry file.
- Load paths must end with `.star`.
- Do not load from `.daiag/tasks/` — tasks are inline.
- Do not create a `lib/` module unless at least two workflow files would share it.

## Validation Checklist

Before finishing, verify all of the following:

- all filenames and workflow IDs use underscores, not dashes
- workflow ID is self-descriptive, unique in `.daiag/workflows/WORKFLOWS.md`, and matches the filename without `.star`
- workflow is placed in `.daiag/workflows/<workflow_id>/`
- single-task workflow uses the task name as the step ID (no qualifier suffix)
- every task helper is defined inline in the `.star` file
- every task helper accepts `step_id` as its first argument
- the task ID is `step_id` directly — no concatenation inside the helper
- prompt file named `<workflow_name>_<task_name>.md` (or `<workflow_name>.md` for single-task)
- prompt files are siblings of the `.star` file
- every `${NAME}` in each `.md` file appears in `vars`
- every JSON key promised in each `.md` file appears in `result_keys`
- every file the prompt says to create or update appears in `artifacts`
- every artifact value is wrapped in `artifact(...)`
- every `path_ref(...)` names an earlier step and a declared artifact key on that step
- every `json_ref(...)` names a declared result key on the referenced step
- `json_ref(...)` inside `until` references a task inside the loop body
- step IDs are globally unique across the entire workflow including loop body tasks
- every task has an executor (own or via `default_executor`)
- `repeat_until` has all four required fields: `id`, `max_iters`, `steps`, `until`
- `max_iters` is at least `1`
- the top-level `wf` variable is assigned and created by `workflow(...)`
- `.daiag/workflows/WORKFLOWS.md` has been updated

## Output

When complete:

- write `.daiag/workflows/<workflow_id>/<workflow_id>.star`
- write one prompt `.md` per task
- update `.daiag/workflows/WORKFLOWS.md`
- give the user a brief summary:
  1. workflow name and parameters
  2. steps in order with their prompt files
  3. any loops and their exit conditions
  4. any subworkflows composed
