---
name: workflow-author
description: Design and author a complete daiag workflow from user requirements. Produces the workflow entry .star file at .daiag/workflows/ and any missing task pairs in .daiag/tasks/. Use when the user wants to create a new workflow or wire up steps into a runnable workflow file.
compatibility: daiag project
---

# Workflow Author

Your job is to turn the user's requirements into a runnable daiag workflow.

Follow the conventions in [`.daiag/agents/workflow-author.md`](.daiag/agents/workflow-author.md).

## Workflow

1. **Load the guide** — Read `.daiag/agents/workflow-author.md` before writing any file.
2. **Clarify requirements** — Ask the questions listed in the guide's "Required Clarifications" section. Do not skip any that are unanswered. Ask as one grouped message, not one question at a time.
3. **Check existing tasks** — List the contents of `.daiag/tasks/` and identify which required steps already have task pairs.
4. **Report missing tasks** — If any required task pair is missing, report them in the structured format from the guide and stop. Do not write the workflow file. Let the user author the missing tasks (e.g. using the `workflow-task-author` skill) and come back.
5. **Write the workflow entry file** — Once all tasks exist, create `.daiag/workflows/<name>.star` wiring all tasks together.
6. **Validate** — Run through the validation checklist in the guide before reporting done.

## What You Produce

- `.daiag/workflows/<name>.star` — the runnable workflow entry file
- A structured missing-tasks report if any required tasks don't exist yet (no files written in that case)

## Key Rules

- **No paths module** — compute paths inline with `format(...)` in the workflow entry file
- **Load from tasks** — always `load("../tasks/<step>.star", "<step>_task")`
- **Suffix** — pass a short suffix like `"main"` to each task helper; tasks inside a loop share the same suffix
- **Loops** — use `repeat_until(...)` when a step must retry until a quality or approval condition; the `until` predicate references a `json_ref` to the last task in the loop body
- **References** — use `path_ref(...)` for file handoff, `json_ref(...)` only for loop exit conditions or small metadata

## Questions to Ask

Before writing files, ask the user:

1. What is the workflow name?
2. What CLI parameters does it accept (`--param key=value`)?
3. What are the steps in order — what does each one do, read, and write?
4. Are any steps iterative? If yes: which tasks form the loop body, what result key and value exit the loop, and how many max iterations?
5. What is the output path pattern for artifacts?
6. What output path pattern should artifacts follow?

## Don't Guess

If the user hasn't specified the workflow name, parameters, steps, or output paths — ask. These shape the entire file and cannot be inferred reliably.
