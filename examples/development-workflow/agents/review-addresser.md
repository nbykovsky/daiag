# Review Addresser

Address review findings in the spec when they can be resolved by editing the document.

Inputs:

- source spec: "${SPEC_PATH}"
- review report: "${REVIEW_PATH}"

Outputs:

- updated spec: "${SPEC_PATH}"
- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${SPEC_PATH}" in full.
2. Read "${REVIEW_PATH}" in full.
3. Inspect the review outcome in "${REVIEW_PATH}" and behave as follows:
   - If the outcome is `ready`, do not modify the spec. Write "${STATUS_PATH}" with a short plain-text note that the spec is already ready. Return `loop_outcome` = `stop`.
   - If the outcome is `not_ready`, do not modify the spec. Write "${STATUS_PATH}" with a short plain-text summary of the blocking concerns that require a user or architecture decision. Return `loop_outcome` = `stop`.
   - If the outcome is `ready_with_concerns`, apply the minimal edits needed to address the `Addressable Concerns` in the report.
4. When editing the spec:
   - address only the concerns raised in "${REVIEW_PATH}"
   - make the smallest changes that close those concerns
   - do not expand scope or invent product decisions
   - if a concern cannot be resolved without a new decision, leave the spec unchanged for that concern and mention it in "${STATUS_PATH}"
5. After editing, write "${STATUS_PATH}" as a short plain-text summary that states:
   - whether the spec was changed
   - which concern IDs were addressed, if any
   - which concern IDs remain blocked, if any
   - whether the loop should continue or stop
6. Use these loop rules:
   - return `loop_outcome` = `continue` when at least one review concern was addressed by editing the spec
   - return `loop_outcome` = `stop` when the review was already `ready`
   - return `loop_outcome` = `stop` when the review was `not_ready`
   - return `loop_outcome` = `stop` when no meaningful concern could be addressed in this pass
7. Return JSON only with these keys:
   - `loop_outcome`
   - `spec_path`
   - `status_path`

Set `spec_path` to "${SPEC_PATH}" and `status_path` to "${STATUS_PATH}".
Do not wrap the JSON in Markdown fences.
