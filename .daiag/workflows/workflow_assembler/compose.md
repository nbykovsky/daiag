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
5. Return a `full_description` — a complete, structured description of the workflow naming the exact catalog workflow IDs to use as subworkflow steps, their order, any loop or conditional control flow with explicit conditions, and how data flows between steps via `path_ref` and `json_ref`. This will be passed directly to `workflow_lifecycle` to implement the workflow.

Outputs:
- Write: `${COMPOSITION_PLAN_PATH}`
- Return JSON with keys:
  - `full_description`: detailed structured description for implementing the workflow, naming catalog workflow IDs as subworkflow steps, their sequencing, and data flow via path_ref and json_ref

Do not wrap the JSON in Markdown fences.
