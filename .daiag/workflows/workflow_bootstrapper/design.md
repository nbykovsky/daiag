# Design Workflow

<!-- DOLLAR{VAR_NAME} is used in this file to describe DOLLAR{VAR_NAME} template placeholders
     in generated prompt files without triggering substitution in this file's own rendering. -->

Inputs:
- `description`: ${DESCRIPTION}
- `workflows_lib`: ${WORKFLOWS_LIB}
- `blueprint_path`: ${BLUEPRINT_PATH}

Instructions:

1. Read `${WORKFLOWS_LIB}/WORKFLOWS.md`. Note existing workflow IDs to avoid collisions and identify reusable workflows.

2. If the description is too ambiguous to implement safely, write `${BLUEPRINT_PATH}` containing:
   - `outcome: needs_clarification`
   - The clarification questions needed before implementation can proceed

   Return `workflow_id` as empty string and `outcome` as `needs_clarification`.

3. Otherwise, choose a unique snake_case workflow ID and write a blueprint to `${BLUEPRINT_PATH}` using this template:

```
outcome: complete
workflow_id: <id>
goal: <one-sentence goal>

## Steps

### <task_id>
Reads: <files or refs this task reads>
Writes: <artifact paths this task writes>
Returns: `<key>`: <description>, ...

## Output Artifacts
- `<key>` — <artifact path>

## Output Results
- `<key>` — <description>
```

Outputs:
- Write: ${BLUEPRINT_PATH}
- Return JSON with keys:
  - `workflow_id`: the planned workflow ID, or empty string when outcome is needs_clarification
  - `outcome`: one of `complete`, `needs_clarification`

Do not wrap the JSON in Markdown fences.
