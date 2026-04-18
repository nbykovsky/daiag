# Compose Workflow Plan

Inputs:
- `description`: ${DESCRIPTION}
- `workflows_lib`: ${WORKFLOWS_LIB}

Instructions:
1. Read `${WORKFLOWS_LIB}/WORKFLOWS.md` to discover the available catalog workflows and their input/output contracts.
2. Analyze the workflow description to identify which catalog workflows should be used as subworkflow steps.
3. Plan the full structure of the workflow including control flow and data flow:
   - Which steps are linear (run once, in order)
   - Which steps belong inside a `repeat_until(...)` loop — identify the loop's `until` condition (e.g. `eq(json_ref("step_id", "outcome"), "complete")`) and `max_iters`
   - Which steps belong inside a `when(...)` conditional branch — identify the condition and whether an `else_steps` branch is needed
   - How data flows between steps via `path_ref` and `json_ref`
4. Write a detailed composition plan to `${COMPOSITION_PLAN_PATH}`. The plan must include:
   - The proposed workflow ID and a one-sentence description
   - A list of steps in execution order, each with:
     - Step type: `task`, `subworkflow`, `repeat_until`, or `when`
     - For `subworkflow` steps: the catalog workflow ID, its inputs, and how those inputs are wired (direct input, `json_ref`, or `path_ref` from an earlier step)
     - For `task` steps: the task's purpose, artifacts written, and result keys returned
     - For `repeat_until` steps: the child steps inside the loop, the `until` condition, and `max_iters`
     - For `when` steps: the condition, the steps in the true branch, and any `else_steps`
   - A data-flow summary showing how outputs from each step feed into subsequent steps
5. The composition plan file is the complete specification that the implement step will read directly — make it precise and unambiguous.

Outputs:
- Write: `${COMPOSITION_PLAN_PATH}`
- Return JSON with keys:
  - `outcome`: `complete`

Do not wrap the JSON in Markdown fences.
