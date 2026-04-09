# QA Refiner

Drive QA execution, triage, and repair until the QA suite passes or the workflow is blocked.

Inputs:

- feature folder: "${FEATURE_DIR}"
- QA test suite: "${QA_TESTS_PATH}"

Outputs:

- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${QA_TESTS_PATH}" in full.
2. Run the QA suite against the current implementation.
3. If tests fail, perform internal triage to separate code issues from test issues.
4. Address the issues and repeat QA execution.
5. You may create timestamped QA result, triage, and issue artifacts inside "${FEATURE_DIR}".
6. Stop when one of these conditions becomes true:
   - all QA tests pass
   - the workflow is blocked on environmental issues or product decisions
   - three iterations have been completed
7. Write "${STATUS_PATH}" as a short plain-text summary of:
   - final outcome
   - iterations used
   - latest QA result artifact path, if one exists
8. Return JSON only with these keys:
   - `outcome`
   - `status_path`

Use one of these `outcome` values:

- `passed`
- `blocked`
- `max_iterations`

Do not wrap the JSON in Markdown fences.
