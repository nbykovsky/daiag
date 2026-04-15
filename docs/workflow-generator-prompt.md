# Workflow Generator Prompt

A prompt for an AI agent that generates complete, runnable `daiag` workflows of
arbitrary complexity from a natural language description.

Replace `${PLACEHOLDER}` with actual values when wiring this into a task.

---

```
# Generate Workflow

Inputs:
- `description`: ${DESCRIPTION}
- `workflows_lib`: ${WORKFLOWS_LIB}
- `output_path`: ${OUTPUT_PATH}

## Role

You generate complete, runnable daiag workflows from a natural language
description. You write:

- `${WORKFLOWS_LIB}/<workflow_id>/workflow.star`
- `${WORKFLOWS_LIB}/<workflow_id>/<task_id>.md` — one per `task(...)` step
- Append an entry to `${WORKFLOWS_LIB}/WORKFLOWS.md`

If the description is too ambiguous to implement safely, write the clarification
questions to `${OUTPUT_PATH}` and return early (see Return section).

---

## DSL Reference

### Structural nodes

| Node | When to use |
|---|---|
| `task(id, prompt, artifacts, result_keys, executor?)` | One agent action with a concrete output |
| `repeat_until(id, max_iters, steps, until)` | A body that retries until a condition |
| `when(id, condition, steps, else_steps?)` | A branch that runs only when a condition holds |
| `subworkflow(id, workflow, inputs)` | Delegate to an existing catalog workflow |

### Value expressions

| Expression | Usable as | When to use |
|---|---|---|
| `"literal"` | string / value | Constant |
| `input(name)` | string / value | Runtime workflow input |
| `format("{x}", x=expr)` | string / value | Interpolate other expressions |
| `run_dir()` | string | Run output directory (absolute) |
| `projectdir()` | string | Project root (absolute) |
| `path_ref(step_id, artifact_key)` | string | Artifact path produced by an earlier step |
| `json_ref(step_id, field)` | value | Result value from an earlier step |
| `loop_iter(loop_id)` | value | Current 1-based iteration (valid only inside that loop) |
| `eq(left, right)` | predicate | Equality — used in `until` and `condition` |

### Visibility rules

- `path_ref` and `json_ref` may only reference steps that appear **earlier in the
  same scope**. They cannot forward-reference.
- Steps inside a `repeat_until` body or a `when` branch are **not visible** from
  the parent scope.
- After a `repeat_until` exits, `path_ref` and `json_ref` to steps inside the loop
  body resolve to the **last iteration's** values.

---

## Pattern Library

### Pattern A — Sequential tasks

Use when each step produces something the next step consumes.

```python
workflow_id = "my_workflow"
topic = input("topic")
draft_path = format("{run_dir}/{wf}/draft.md", run_dir = run_dir(), wf = workflow_id)
summary_path = format("{run_dir}/{wf}/summary.md", run_dir = run_dir(), wf = workflow_id)

wf = workflow(
    id = workflow_id,
    inputs = ["topic"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "write_draft",
            prompt = template_file("write_draft.md", vars = {
                "TOPIC": topic,
                "DRAFT_PATH": draft_path,
            }),
            artifacts = {"draft": artifact(draft_path)},
            result_keys = ["draft_path"],
        ),
        task(
            id = "summarize",
            prompt = template_file("summarize.md", vars = {
                "DRAFT_PATH": path_ref("write_draft", "draft"),
                "SUMMARY_PATH": summary_path,
            }),
            artifacts = {"summary": artifact(summary_path)},
            result_keys = ["summary_path"],
        ),
    ],
    output_artifacts = {
        "draft": path_ref("write_draft", "draft"),
        "summary": path_ref("summarize", "summary"),
    },
    output_results = {"summary_path": json_ref("summarize", "summary_path")},
)
```

### Pattern B — Retry loop (single task body)

Use when one task must repeat until an approval or quality condition is met.

```python
review_path = format("{run_dir}/{wf}/review_{iter}.md",
    run_dir = run_dir(), wf = workflow_id,
    iter = loop_iter(loop_id = "review_loop"))

repeat_until(
    id = "review_loop",
    max_iters = 4,
    steps = [
        task(
            id = "review",
            prompt = template_file("review.md", vars = {
                "DRAFT_PATH": path_ref("write_draft", "draft"),
                "REVIEW_PATH": review_path,
            }),
            artifacts = {"review": artifact(review_path)},
            result_keys = ["outcome"],
        ),
    ],
    until = eq(json_ref("review", "outcome"), "approved"),
)
```

After the loop, `path_ref("review", "review")` gives the last iteration's report.
Use `loop_iter(...)` inside `format(...)` whenever the loop produces multiple
artifacts that must not overwrite each other.

### Pattern C — Revise-then-review loop (multi-task body)

Use when each iteration must both revise an artifact and then assess it.

```python
review_path = format("{run_dir}/{wf}/review_{iter}.md",
    run_dir = run_dir(), wf = workflow_id,
    iter = loop_iter(loop_id = "improve_loop"))

repeat_until(
    id = "improve_loop",
    max_iters = 3,
    steps = [
        task(
            id = "revise",
            prompt = template_file("revise.md", vars = {
                "DRAFT_PATH": path_ref("write_draft", "draft"),
            }),
            artifacts = {"draft": artifact(path_ref("write_draft", "draft"))},
            result_keys = ["draft_path"],
        ),
        task(
            id = "assess",
            prompt = template_file("assess.md", vars = {
                "DRAFT_PATH": path_ref("revise", "draft"),
                "REPORT_PATH": review_path,
            }),
            artifacts = {"report": artifact(review_path)},
            result_keys = ["outcome"],
        ),
    ],
    until = eq(json_ref("assess", "outcome"), "ready"),
)
```

The `until` predicate must reference a result key from a task **inside** the loop
body, and that task must be the last step that sets the key in every iteration.

### Pattern D — Conditional branch

Use when a step should only run if a prior result matches a specific value.

```python
when(
    id = "fix_if_broken",
    condition = eq(json_ref("validate", "outcome"), "broken"),
    steps = [
        task(
            id = "fix",
            prompt = template_file("fix.md", vars = {...}),
            artifacts = {"fixed": artifact(fixed_path)},
            result_keys = ["outcome"],
        ),
    ],
    else_steps = [],
)
```

Steps inside `when` are not visible to later sibling steps or workflow outputs.
Do not reference them with `path_ref` or `json_ref` outside the branch.

### Pattern E — Subworkflow composition

Use when an existing catalog workflow should be reused as a stage.

```python
subworkflow(
    id = "refine",
    workflow = "spec_refinement",
    inputs = {
        "feature_dir": feature_dir,
        "spec_path": spec_path,
    },
)
# Parent reads child outputs:
path_ref("refine", "spec")
json_ref("refine", "outcome")
```

Read `${WORKFLOWS_LIB}/WORKFLOWS.md` first to discover available catalog workflows
and their declared input/output contracts.

### Pattern F — Mixed: sequential → loop → conditional

Wire patterns together in order, passing artifacts downstream:

```python
steps = [
    task(id = "parse", ...),                    # A: produce initial artifact
    repeat_until(id = "refine_loop", ...),      # B: refine until good enough
    when(id = "escalate_if_blocked", ...),      # D: conditional follow-up
]
```

The parent workflow can only reference steps at its **own scope**:
`path_ref("parse", ...)`, `path_ref("refine_loop_inner_task", ...)` is invalid
from the parent — use a result from the loop's last body task via `json_ref` to
route, and surface what you need through `output_artifacts` / `output_results` of
a subworkflow if isolation is required.

---

## Reasoning Process

Work through the following before writing any file:

1. **Inputs.** What values does the caller supply? These become `input(name)`
   references and the `inputs = [...]` list.

2. **Step classification.** For each conceptual step, assign it a kind:
   - `task` — a single agent action
   - `repeat_until` — a body that must retry
   - `when` — a branch that runs conditionally
   - `subworkflow` — an existing catalog workflow

3. **Loop bodies.** For each `repeat_until`, identify:
   - Which tasks form the body (often: revise + assess)?
   - Which result key drives the exit condition and what is the exit value?
   - Do loop iterations write artifacts that must be preserved across iterations?
     If yes, use `loop_iter(loop_id = "...")` inside a `format(...)` path.

4. **Conditional triggers.** For each `when`, identify:
   - Which step's result drives the condition?
   - What value triggers the branch?
   - Is an else branch needed?

5. **Data flow.** For each step, trace:
   - Which earlier artifacts does it read? (`path_ref(...)`)
   - Which result values does it consume? (`json_ref(...)`)
   - What does it write and return?

6. **Outputs.** Which artifacts and results should the workflow expose to callers?

7. **Dependency check.** Confirm every `path_ref` and `json_ref` targets a step
   that is visible from the call site (same scope, earlier in the list).

---

## File Rules

### workflow.star

```python
workflow_id = "<id>"
<var> = input("<name>")
# Compute derived paths at the top when the same path is used in multiple tasks.
<artifact_path> = format("{run_dir}/{wf}/<name>.<ext>",
    run_dir = run_dir(), wf = workflow_id)

wf = workflow(
    id = workflow_id,
    inputs = ["<name>", ...],   # [] when no inputs
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [...],
    output_artifacts = {"<key>": path_ref("<step_id>", "<artifact_key>")},
    output_results = {"<key>": json_ref("<step_id>", "<result_key>")},
)
```

- `wf` must be assigned at top level.
- Use `run_dir()` for all generated output paths.
- Use `projectdir()` only when referencing a project source file.
- Every artifact value must be wrapped in `artifact(...)`.
- Declare both `output_artifacts` and `output_results`; use `{}` when empty.
- Task IDs use snake_case. Add a qualifier only when the same role appears twice.

### Prompt template files (`<task_id>.md`)

```markdown
# <Task Title>

Inputs:
- `<name>`: ${PLACEHOLDER}

Instructions:
1. <exact, numbered step>
2. ...

Outputs:
- Write/update: ${ARTIFACT_PATH}
- Return JSON with keys:
  - `<key>`: <description>
  - `outcome`: one of `<value_a>`, `<value_b>`

Do not wrap the JSON in Markdown fences.
```

- Every dynamic value uses `${PLACEHOLDER}` with a matching `vars` entry in
  `workflow.star`.
- Specify file-edit semantics explicitly: replace / append / update in place.
- Every JSON key listed must appear in `result_keys`. List all enum values for
  fields like `outcome`.
- Every output file listed must have a matching `artifacts` entry.

---

## Instructions

1. Read `${WORKFLOWS_LIB}/WORKFLOWS.md`. Note existing IDs to avoid collisions
   and identify any reusable catalog workflows.

2. If the description is ambiguous, write the clarification questions to
   `${OUTPUT_PATH}` and stop. Return `outcome: needs_clarification`.

3. Apply the Reasoning Process above to plan the workflow structure.

4. Write `${WORKFLOWS_LIB}/<workflow_id>/workflow.star`.

5. Write one `<task_id>.md` per `task(...)` step (not per `subworkflow`).

6. Append to `${WORKFLOWS_LIB}/WORKFLOWS.md`:

   ```
   ## <workflow_id>

   <one-sentence description>

   File: `${WORKFLOWS_LIB}/<workflow_id>/workflow.star`

   Inputs:
   - `<input>` — <description>

   Output Artifacts:
   - `<key>` — <description>

   Output Results: `<key1>`, `<key2>`, ...
   ```

7. Write a brief summary to `${OUTPUT_PATH}`:
   - Workflow ID and path to `workflow.star`
   - Steps in order with their kinds and prompt files
   - Any loops with their exit conditions
   - Any conditions with their trigger values
   - Any catalog subworkflows composed

---

## Return

Return JSON with keys:
- `workflow_id`: the generated workflow ID, or `""` on needs_clarification
- `workflow_path`: absolute path to `workflow.star`, or `""` on needs_clarification
- `outcome`: one of `complete`, `needs_clarification`

Do not wrap the JSON in Markdown fences.
```
