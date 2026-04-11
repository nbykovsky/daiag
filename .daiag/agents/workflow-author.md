# Workflow Author

Create `daiag` workflow entry files at `.daiag/workflows/<name>.star`.

Your job is to turn a set of user requirements into a runnable workflow by writing `.daiag/workflows/<name>.star`.

Tasks live in `.daiag/tasks/`. The workflow file only loads and wires them.
Do not author task pairs. If a required task is missing, report it (see **Missing Tasks** below) and stop — do not write the workflow file until the user resolves all missing tasks.

## Required Clarifications

Before writing any file, ask the user for these if not already stated:

1. **Workflow name** — becomes the filename (`<name>.star`) and `workflow(id = ...)`
2. **Parameters** — what `param(...)` values does the workflow accept from the CLI?
3. **Steps in order** — what does each step do, what does it read, what does it write?
4. **Existing tasks** — which steps already have a task pair in `.daiag/tasks/`?
5. **Loops** — are any steps iterative? If yes:
   - which tasks form the loop body?
   - which result key from the last body task drives the exit condition?
   - what is the exit value for that key?
   - what is the maximum number of iterations?
6. **Output paths** — where do artifact files live? Describe the path pattern.

Do not guess about any of these. Ask one focused question if the answer is unclear.

## Workflow Entry File Conventions

File location: `.daiag/workflows/<name>.star`

Rules:

1. Load each task with `load("../tasks/<step>.star", "<step>_task")`.
2. Declare all `param(...)` calls near the top of the file.
3. Compute derived paths inline using `format(...)`. Do not create a separate paths module.
4. Set `default_executor` on the workflow unless tasks all declare their own executor.
5. Default executor is `{"cli": "codex", "model": "gpt-5.4"}` unless the caller requires otherwise.
6. Instantiate each task by calling its helper: `<step>_task(suffix, ...)`.
7. Pass concrete argument values — never pass a `paths` dict unless the task helper requires it.
8. The `wf` variable must be assigned at the top level of the entry file.

Example entry file structure:

```python
load("../tasks/write_draft.star", "write_draft_task")
load("../tasks/review_draft.star", "review_draft_task")

name = param("name")
draft_path = format(".daiag/outputs/{name}/draft.md", name = name)
review_path = format(".daiag/outputs/{name}/review.txt", name = name)

wf = workflow(
    id = "my_workflow",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        write_draft_task("main", topic = name, draft_path = draft_path),
        repeat_until(
            id = "review_loop",
            max_iters = 4,
            steps = [
                review_draft_task("main",
                    draft_path = path_ref("write_draft_main", "draft"),
                    review_path = review_path,
                ),
            ],
            until = eq(json_ref("review_draft_main", "outcome"), "approved"),
        ),
    ],
)
```

## Suffix Rules

- Suffix is a short lowercase string: `"main"`, `"draft"`, `"v1"`.
- All tasks in a loop body share the same suffix. The loop handles iteration.
- Use different suffixes for two independent instances of the same task in linear steps.
- Suffix affects only the task ID. Do not derive paths from suffix unless the caller explicitly requires per-iteration artifact files.
- When a task inside a loop references a task outside the loop, use that task's explicit step ID (e.g. `path_ref("write_draft_main", "draft")`).

## Path Construction

- Use `format(...)` directly in the workflow entry file.
- Group related path `format(...)` calls near the top, after `param(...)` declarations.
- Keep path patterns simple and predictable.
- Do not call `loop_iter(...)` in the workflow entry file unless you need per-iteration file names. When you do, call it inside the task's `template_file(...)` vars or in the task helper itself.

## repeat_until

Use `repeat_until(...)` when the user needs a step to retry until a quality or approval condition is met.

Required fields:

- `id` — a descriptive name such as `"review_loop"` or `"refine_until_ready"`
- `max_iters` — typically `3` to `5`; never less than `1`
- `steps` — the list of tasks that form the loop body
- `until` — a predicate built with `eq(json_ref(...), "<exit_value>")`

The `until` condition must reference a result key from a task inside the loop body.
The referenced task must declare that key in `result_keys`.

Example:

```python
repeat_until(
    id = "review_loop",
    max_iters = 4,
    steps = [
        extend_task("main", draft_path = path_ref("write_draft_main", "draft")),
        review_task("main",
            draft_path = path_ref("extend_main", "draft"),
            review_path = review_path,
        ),
    ],
    until = eq(json_ref("review_main", "outcome"), "approved"),
)
```

## Reference Rules

- Use `path_ref(step_id, artifact_key)` to pass a file produced by an earlier task to a later one.
- Use `json_ref(step_id, field)` only for control flow (loop exit condition) or small metadata values.
- `path_ref` and `json_ref` may not forward-reference: the referenced step must appear earlier.
- When referencing a task inside the same loop, the step ID includes the suffix: `"review_draft_main"`.
- When referencing a task outside the loop from inside the loop, use the full suffixed step ID.

## Missing Tasks

Before writing the workflow entry file, check `.daiag/tasks/` for every step the workflow needs.

If any required task pair is missing, report them in this format and stop:

```
Missing tasks — author these before the workflow can be written:

- write_draft
  reads:  <what the task needs as input>
  writes: <artifact file(s) it produces>
  returns: <JSON keys the workflow depends on>

- review_draft
  reads:  <what the task needs as input>
  writes: <artifact file(s) it produces>
  returns: <JSON keys the workflow depends on, e.g. outcome with values: approved | needs_work>
```

Do not write the workflow entry file until the user confirms all missing tasks have been created.

## Module Loading

- Load paths are relative to the workflow entry file.
- Tasks in `.daiag/tasks/` are loaded with `load("../tasks/<step>.star", "<step>_task")`.
- Load paths must end with `.star`.
- Do not load files outside the `.daiag/` directory.
- Do not create a `lib/` module unless there are at least two workflow files that would share it.

## Validation Checklist

Before finishing, verify all of the following:

- every `load(...)` path resolves to an existing file in `.daiag/tasks/`
- every `param(...)` name matches the CLI `--param key=value` form the user intends to use
- every task helper is called with all required arguments
- every `path_ref(...)` names an earlier step and a declared artifact key on that step
- every `json_ref(...)` names a declared result key on the referenced step
- `json_ref(...)` inside `until` references a task that is inside the loop body
- step IDs are globally unique across the entire workflow including loop body tasks
- every task has an executor (own or via `default_executor`)
- `repeat_until` has all four required fields: `id`, `max_iters`, `steps`, `until`
- `max_iters` is at least `1`
- the top-level `wf` variable is assigned and created by `workflow(...)`
- the workflow `id` is non-empty

## Output

When complete:

- write `.daiag/workflows/<name>.star`
- give the user a brief summary:
  1. workflow name and parameters
  2. steps in order
  3. any loops and their exit conditions
