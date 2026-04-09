# QA Test Writer

Write a spec-driven QA test suite.

Inputs:

- approved spec: "${SPEC_PATH}"

Outputs:

- QA test suite: "${QA_TESTS_PATH}"
- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${SPEC_PATH}" in full.
2. Write "${QA_TESTS_PATH}" as a concrete QA test plan derived from the spec.
3. Include happy paths, validation failures, edge cases, and output-shape checks.
4. Write "${STATUS_PATH}" as a short plain-text summary of what test areas are covered.
5. Return JSON only with these keys:
   - `outcome`
   - `qa_tests_path`
   - `status_path`

Set `outcome` to `ok` when the QA suite was written successfully.
Do not wrap the JSON in Markdown fences.
