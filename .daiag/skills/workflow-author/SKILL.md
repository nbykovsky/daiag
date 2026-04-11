---
name: workflow-author
description: Design and author a complete daiag workflow from user requirements. Produces the workflow .star file with inline task definitions and sibling prompt templates. Use when the user wants to create a new workflow, author tasks, or wire up steps into a runnable workflow file.
compatibility: daiag project
---

# Workflow Author

Your job is to turn the user's requirements into a runnable daiag workflow.

Follow the conventions in [`.daiag/agents/workflow-author.md`](.daiag/agents/workflow-author.md).

## Workflow

1. **Load the guide** — Read `.daiag/agents/workflow-author.md` before writing any file.
2. **Clarify requirements** — Ask the questions listed in the guide's "Required Clarifications" section. Do not skip any that are unanswered. Ask as one grouped message, not one question at a time.
3. **Read the workflow index** — Read `.daiag/WORKFLOWS.md` to discover available workflows and their input/output contracts.
4. **Write the workflow file** — Create `<dir>/<workflow_name>.star` with inline task definitions and sibling prompt templates.
5. **Update the index** — Add or update the entry in `.daiag/WORKFLOWS.md`.
6. **Validate** — Run through the validation checklist in the guide before reporting done.

## What You Produce

- `<dir>/<workflow_name>.star` — the runnable workflow entry file with inline tasks
- `<dir>/<workflow_name>_<task_name>.md` per task (or `<workflow_name>.md` for single-task workflows)
- Updated `.daiag/WORKFLOWS.md`

## Key Rules

- **Inline tasks** — define task helpers directly in the `.star` file, do not load from `.daiag/tasks/`
- **Underscores everywhere** — filenames, workflow IDs, and task names all use underscores
- **Prompt files are siblings** — prompt `.md` files live in the same directory as the `.star` file
- **Sharing via subworkflow** — any workflow can be reused as a subworkflow; use `workflow(inputs = [...])` and declare `output_artifacts`/`output_results`
- **No paths module** — compute paths inline with `format(...)` in the workflow entry file
- **Step ID** — pass the full step ID to each task helper (e.g. `"write_draft_main"`); the same string is used in `path_ref(...)` and `json_ref(...)`
- **Loops** — use `repeat_until(...)` when a step must retry until a quality or approval condition; the `until` predicate references a `json_ref` to the last task in the loop body

## Questions to Ask

Before writing files, ask the user:

1. What is the workflow ID? (becomes the filename and `workflow(id = ...)` — use underscores)
2. What are the steps in order — what does each one do, read, and write?
3. Are any steps iterative? If yes: which tasks form the loop body, what result key and value exit the loop, and how many max iterations?
4. What is the output path pattern for artifacts?
5. Is this workflow intended to be reused as a subworkflow?

## Don't Guess

If the user hasn't specified the workflow ID, steps, or output paths — ask. These shape the entire file and cannot be inferred reliably.
