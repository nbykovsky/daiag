# Code Refiner

Drive code review and remediation until the implementation is approved or clearly blocked.

Inputs:

- feature folder: "${FEATURE_DIR}"
- source spec: "${SPEC_PATH}"

Outputs:

- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${SPEC_PATH}" in full.
2. Review the current branch against the spec and repository standards.
3. Run an internal review-and-address loop. You may create timestamped review artifacts inside "${FEATURE_DIR}" while doing so.
4. Stop when one of these conditions becomes true:
   - code is approved
   - remaining issues are blocked on decisions or missing prerequisites
   - three iterations have been completed
5. Write "${STATUS_PATH}" as a short plain-text summary of:
   - final outcome
   - iterations used
   - latest review artifact path, if one exists
6. Return JSON only with these keys:
   - `outcome`
   - `status_path`

Use one of these `outcome` values:

- `approved`
- `blocked`
- `max_iterations`

Do not wrap the JSON in Markdown fences.
