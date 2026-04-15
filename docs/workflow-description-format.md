# Workflow Description Format

## Strategy

Generating complex workflows reliably requires two steps:

1. **Describe** — an AI reads the user's request and the current catalog
   (`WORKFLOWS.md`), then writes a natural-language workflow description.
   It identifies which stages can reuse catalog workflows, which need new tasks,
   and what is ambiguous or missing. The output is a human-readable document
   that a human can read, edit, and confirm before code is written.

2. **Construct** — a second AI reads the description and `WORKFLOWS.md`, then
   writes the `.star` entry file and prompt templates. It only maps description
   concepts to DSL; it does not redesign the workflow.

This split keeps the design decision (what the workflow should do and how) separate
from the implementation decision (how to express it in Starlark). Gaps caught in
step 1 cost nothing; gaps caught in step 2 require re-generating code.

---

## Format

A workflow description is a Markdown document with five sections.

### 1. Header

```
# Workflow: <workflow_id>

Goal: <one sentence describing what the workflow achieves>
```

`workflow_id` must be unique in the catalog and use snake_case.

### 2. Inputs

```
## Inputs

- `<name>` — <what the caller supplies>
```

List every value the caller must supply at runtime. Omit if there are none.

### 3. Steps

```
## Steps
```

Steps are numbered in execution order. Each step has a **kind tag** and a
consistent set of sub-items.

#### Task (new)

```
### N. <name>  [task]

Reads:
- `<input name>` input
- `<artifact key>` from step M (<step name>)

Writes:
- `<workflow_id>/<file>.<ext>` — <one-line description>

Returns:
- `outcome`: one of `<value_a>`, `<value_b>`
- `<result_key>`: <description>
```

Use `[task]` when no catalog workflow covers this step. The `Writes:` section
becomes the task's `artifacts`. The `Returns:` section becomes `result_keys`.

#### Subworkflow (catalog)

```
### N. <name>  [subworkflow: <catalog_workflow_id>]

Inputs bound:
- `<workflow_input>` ← `<source input or artifact>`

Exposes:
- artifact `<key>` — <description>
- result `<key>` — <description>
```

Use `[subworkflow: <id>]` when an existing catalog workflow covers this step
exactly. Copy the input names from `WORKFLOWS.md`.

#### Loop

```
### N. <loop name>  [loop, max <N> iterations, until <inner step>.<result_key> == "<value>"]

Body:

#### N.a <name>  [task]
...

#### N.b <name>  [task]
...

Per-iteration artifacts: yes / no
```

The body steps are formatted like regular tasks or subworkflows but indented one
heading level. Mark `Per-iteration artifacts: yes` when the loop writes files
that must not be overwritten between iterations (e.g. reports, reviews).

#### Conditional

```
### N. <name>  [when <step M>.<result_key> == "<value>"]

Then:
- <step reference or inline task>

Else:
- (none) / <step reference or inline task>
```

Steps inside a conditional are not visible to later sibling steps. Only expose
values to the parent through an inline task's `output_artifacts` /
`output_results`, or note in Gaps that a catalog subworkflow wrapping the branch
is needed.

### 4. Gaps

```
## Gaps

- [missing catalog] <description of what catalog workflow is needed and why>
- [ambiguous] <description of what is unclear and what decision is needed>
- [decision] <trade-off that a human or the caller needs to resolve>
```

The Describe step must populate this section honestly. An empty Gaps section means
the description is complete enough for the Construct step to proceed without
questions.

### 5. Outputs

```
## Outputs

Artifacts:
- `<key>` — `<path>` (from step N)

Results:
- `<key>` — <description> (from step N)
```

---

## Example: `spec_writer`

A workflow that takes a rough feature idea, writes a specification, then refines
it through a review-and-address loop until it is ready or the iteration limit
is reached.

```
# Workflow: spec_writer

Goal: Produce a reviewed, self-consistent feature specification from a rough idea.

## Inputs

- `feature_name` — short identifier for the feature (used in file paths)
- `rough_idea` — a paragraph describing what the feature should do

## Steps

### 1. Write initial spec  [task]

Reads:
- `feature_name` input
- `rough_idea` input

Writes:
- `spec_writer/{feature_name}/spec.md` — the initial draft specification

Returns:
- `outcome`: one of `complete`, `needs_clarification`
- `spec_path`: absolute path to the written spec

---

### 2. Refine spec  [loop, max 3 iterations, until `address_review`.`loop_outcome` == "stop"]

Per-iteration artifacts: yes

#### 2.a. Review spec  [task]

Reads:
- `spec` artifact from step 1 (Write initial spec)

Writes:
- `spec_writer/{feature_name}/review_{iter}.md` — per-iteration review report
  (the iteration suffix prevents overwriting earlier reviews)

Returns:
- `outcome`: one of `approved`, `changes_requested`
- `review_path`: path to the review report

#### 2.b. Address review  [task]

Reads:
- `spec` artifact from step 2.a (Review spec)
- `review_report` artifact from step 2.a (Review spec)

Writes:
- `spec_writer/{feature_name}/spec.md` — updates spec in place

Returns:
- `loop_outcome`: one of `continue`, `stop`
- `spec_path`: path to the updated spec

---

### 3. Escalate if blocked  [when `address_review`.`loop_outcome` == "blocked"]

Then:
- Inline task: write a brief escalation note to `spec_writer/{feature_name}/blocked.md`
  explaining what is unclear and what a human needs to decide.
  Returns: `escalation_path`

Else:
- (none)

## Gaps

- [ambiguous] Step 1 returns `needs_clarification` but there is no conditional
  to halt before the loop. Decide: should the loop run anyway (wasting iterations)
  or should a `when` gate the loop on step 1's outcome?
- [decision] Step 3's inline task is not a catalog workflow. If escalation logic
  grows, it should be extracted to a catalog subworkflow.
- [missing catalog] No existing catalog workflow reviews a spec document. Step 2.a
  needs a new task. If a generic `document_reviewer` catalog workflow is added
  later, step 2.a can be replaced with `[subworkflow: document_reviewer]`.

## Outputs

Artifacts:
- `spec` — `spec_writer/{feature_name}/spec.md` (from step 2.b)
- `last_review` — `spec_writer/{feature_name}/review_{iter}.md` (from step 2.a,
  last iteration)

Results:
- `outcome` — `loop_outcome` from step 2.b (last iteration)
```

---

## Notes for the Describe step

- Read `WORKFLOWS.md` before writing the description. For each step, check whether
  an existing catalog workflow covers it. Prefer `[subworkflow: <id>]` over
  `[task]` when the catalog workflow's input/output contract matches.
- Use `[task]` for any step with no catalog match. Do not invent a catalog
  workflow ID that does not exist.
- Write one gap entry for every decision that would change the structure of the
  workflow. Leave gaps empty only when the description is genuinely unambiguous.
- Do not write `.star` syntax in the description. The Construct step handles
  translation.

## Notes for the Construct step

- A `[task]` step becomes a `task(...)` in `workflow.star` plus a sibling
  `<step_name>.md` prompt template.
- A `[subworkflow: <id>]` step becomes a `subworkflow(...)` with inputs bound
  as described. No new prompt file is needed.
- A `[loop]` with `Per-iteration artifacts: yes` uses `loop_iter(loop_id=...)`
  inside `format(...)` for those artifact paths.
- A `[when]` with an inline then-task emits both a `when(...)` and an inline
  `task(...)` inside it.
- Gaps listed as `[ambiguous]` must be resolved before generating code. If any
  remain unresolved, return `outcome: needs_clarification`.
