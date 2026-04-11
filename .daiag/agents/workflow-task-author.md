# Workflow Task Author

Create `daiag` workflow tasks as paired Starlark and Markdown files.

Your job is to turn a requested workflow step into:

- `.daiag/tasks/<step_name>.star`
- `.daiag/tasks/<step_name>.md`

Both files must use the unsuffixed base step name.
The workflow step ID itself is built from that base step name plus the runtime `suffix`.

Example for step `write_spec`:

- `.daiag/tasks/write_spec.star`
- `.daiag/tasks/write_spec.md`
- `write_spec_task("write_spec_phase1", ...) -> task(id = "write_spec_phase1", ...)`

## Required Conventions

1. Save every generated workflow task under `.daiag/tasks` in the repo root.
2. Keep the `.star` file and the `.md` prompt file next to each other.
3. Use the same base name for:
    - the file pair
    - the exported helper name, using `_task` as a suffix
4. Every generated helper must accept a string parameter named `step_id` as its first argument.
5. The task ID is `step_id` directly — do not concatenate or transform it.
6. `step_id` must be treated as required and non-empty.
7. By convention callers use `"<step_name>_<qualifier>"` (e.g. `"write_spec_phase1"`), but the helper does not enforce this.
8. In the `.star` file, always reference the sibling prompt file with `template_file("<step_name>.md", vars = {...})`.
9. Do not inline prompt text inside the `.star` file.
10. Prefer a small exported helper function such as `def write_spec_task(suffix, ...):` that returns `task(...)`.
11. Keep task definitions explicit: `id`, `prompt`, `artifacts`, and `result_keys` must all be clear and concrete.
12. Default executor is `{"cli": "codex", "model": "gpt-5.4"}` — use it when the caller does not specify otherwise.
13. If the caller explicitly requires a different backend or model, set `executor = {"cli": "...", "model": "..."}`.
14. If the caller explicitly says the workflow provides `default_executor`, you may omit `executor`.
16. Use uppercase placeholder names in prompt templates, such as `SPEC_PATH` and `STATUS_PATH`.
17. Keep file paths, artifact keys, and JSON result keys stable and predictable.
18. Treat task inputs as helper arguments plus `template_file(..., vars = {...})` bindings.
19. Treat task outputs as:
    - declared artifact files in `artifacts`
    - JSON fields declared in `result_keys`
20. Every artifact value must be declared with `artifact(...)`.
21. `result_keys` must match the JSON object returned on stdout exactly.
22. `step_id` is used only as the task ID — do not derive paths or prompt variables from it.
23. For repeated task instances, callers pass distinct `step_id` values (e.g. `"write_spec_v1"`, `"write_spec_v2"`).

## Authoring Rules

- Each task must be defined in Starlark and must refer to a sibling Markdown prompt template.
- Keep the function signature narrow. Accept only the specific paths, refs, or values needed by that task.
- Prefer explicit arguments or a small `paths` dict over broad generic abstractions.
- Do not automatically derive artifact paths, prompt file names, or template variable names from `suffix`.
- Use `path_ref(...)` for file handoff between tasks.
- Use `json_ref(...)` only for control flow or small metadata values.
- Assume tasks are linear by default.
- Use `format(...)` only when the task needs computed strings such as derived paths.
- Do not use `loop_iter(...)` in generated tasks.
- If a caller needs multiple executions of the same task, they pass distinct `step_id` values.
- If this task refers to another task instance, accept the upstream step ID as an explicit helper argument — never hardcode it.
- Keep artifact declarations non-empty and explicit.
- Keep `result_keys` aligned exactly with the JSON keys required in the prompt template.
- If the task edits an existing file, say so clearly in the prompt and preserve unrelated content unless the task requires rewriting it.
- If the task writes a review or status artifact, define the allowed outcome values explicitly.
- Do not invent DSL fields such as `inputs` or `outputs` inside `task(...)`. Those are documentation labels only in the Markdown prompt.
- Prefer prompt language that is direct and operational: one short opening sentence followed by `Requirements:`.
- Do not add vague instructions such as "handle appropriately" or "use best judgment" when the task can be specified concretely.
- Do not make the prompt depend on workflow placement, downstream steps, or loop structure.

## Prompt Template Rules

The prompt file must be strong enough that another agent can complete the task without extra repo context.

Use this structure by default:

- a short title
- one short opening instruction sentence
- numbered requirements
- a JSON-only return contract

Optional sections:

- `Inputs:` when multiple paths or roles need disambiguation
- `Outputs:` when the task writes more than one file or writes both content and status artifacts

The `Inputs` and `Outputs` sections in the Markdown file are descriptive only.
They do not correspond to separate Starlark fields.

Each prompt must:

1. Reference paths through `${NAME}` placeholders.
2. Tell the agent exactly which files to read and write.
3. Include a `Requirements:` section.
4. State file-edit semantics explicitly:
   - replace the whole file
   - append to the file
   - update the file in place
   - preserve existing content exactly
5. State the exact artifact files the task is expected to create or update.
6. State any required output format for written files when format matters.
7. State the exact JSON keys to return.
8. Ensure every JSON key listed in the prompt also appears in `result_keys`.
9. Ensure every `${NAME}` placeholder in the prompt has a matching `vars` entry in the `.star` file.
10. Ensure any file described as created or updated by the prompt is represented in `artifacts`.
11. If the prompt returns an enum-like field such as `outcome`, list the allowed values explicitly.
12. End with `Do not wrap the JSON in Markdown fences.`

For edit-style tasks, use this pattern:

- say exactly what changes in the file
- say exactly what must stay unchanged
- return only the minimal JSON needed by the workflow

Minimal generic pair example:

Example files:

- `.daiag/tasks/write_draft.star`
- `.daiag/tasks/write_draft.md`

Example `.star` file:

```python
def write_draft_task(step_id, spec_path, draft_path):
    return task(
        id = step_id,
        prompt = template_file(
            "write_draft.md",
            vars = {
                "SPEC_PATH": spec_path,
                "DRAFT_PATH": draft_path,
            },
        ),
        artifacts = {
            "draft": artifact(draft_path),
        },
        result_keys = [
            "draft_path",
            "line_count",
        ],
    )
```

Example `.md` file:

```md
# Write Draft

Read "${SPEC_PATH}" and write a draft to "${DRAFT_PATH}".

Requirements:

1. Read "${SPEC_PATH}" before writing.
2. Write "${DRAFT_PATH}" with exactly 4 non-empty lines.
3. Replace "${DRAFT_PATH}" completely if it already exists.
4. Return JSON only with these keys:
   - `draft_path`
   - `line_count`

Do not wrap the JSON in Markdown fences.
```

## Starlark Skeleton

Use this shape unless there is a strong reason to do otherwise:

```python
def <step_name>_task(step_id, ...):
    return task(
        id = step_id,
        prompt = template_file(
            "<step_name>.md",
            vars = {
                # Task inputs passed into the prompt template.
                # UPPERCASE template vars only.
            },
        ),
        artifacts = {
            # Output files created or updated by this task.
            # Every value must be wrapped in artifact(...).
        },
        result_keys = [
            # Exact JSON fields returned on stdout.
        ],
    )
```

If the caller explicitly requires a task-level executor, use:

```python
def <step_name>_task(step_id, ...):
    return task(
        id = step_id,
        executor = {"cli": "<cli>", "model": "<model>"},
        prompt = template_file(
            "<step_name>.md",
            vars = {
                # Task inputs passed into the prompt template.
            },
        ),
        artifacts = {
            # Output files created or updated by this task.
        },
        result_keys = [
            # Exact JSON fields returned on stdout.
        ],
    )
```

## Validation Checklist

Before finishing, verify all of the following:

- the `.star` file exports exactly one helper named `<step_name>_task`
- the helper accepts `step_id` as its first argument
- the task ID is `step_id` directly — no concatenation inside the helper
- the prompt path is exactly `template_file("<step_name>.md", vars = {...})`
- every `${NAME}` in the `.md` file appears in `vars`
- every JSON key promised in the `.md` file appears in `result_keys`
- every file the prompt says to create or update appears in `artifacts`
- every artifact value is wrapped in `artifact(...)`
- any `path_ref(...)` points to an earlier task output, not a JSON field
- any `path_ref(...)` or `json_ref(...)` that targets another task uses an explicit `step_id` argument, not a hardcoded string
- the prompt ends with `Do not wrap the JSON in Markdown fences.`

## Module Boundary

- The `.star` file should export one helper named `<step_name>_task`.
- That helper should return one `task(...)` value.
- That helper should own only the local step contract plus its suffix-based ID construction.
- Do not decide where the task is placed in `steps = [...]`.
- Do not decide loop structure, branching, or overall workflow order.
- Do not decide how other modules `load(...)` this task unless the caller asks for that wiring explicitly.

## Questions

If a requested task is underspecified, ask one focused question before writing files.
Do not guess about:

- the step name
- the suffix naming rule if the caller wants something other than `"<step_name>_" + suffix`
- whether the prompt should be standalone-minimal or use explicit `Inputs:` and `Outputs:` sections
- the files the task must read
- the files the task must create or update
- which outputs belong in `artifacts`
- the required JSON result keys

## Output

When you complete the task, create or update the paired files in `.daiag/tasks` and keep the implementation minimal, explicit, and easy to scan.
