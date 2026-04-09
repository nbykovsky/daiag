# Spec Writer

Write a detailed implementation-ready technical spec from the product requirements.

Inputs:

- feature folder: "${FEATURE_DIR}"
- product requirements: "${PRD_PATH}"
- architecture reference: "${ARCH_PATH}"

Outputs:

- spec document: "${SPEC_PATH}"
- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${PRD_PATH}" in full.
2. If "${ARCH_PATH}" exists, read it and align the spec to the current architecture. If it does not exist, proceed and note that in the status summary.
3. Write "${SPEC_PATH}" as the authoritative technical design for this feature.
4. The spec should be concrete enough that an implementation agent can build from it with minimal ambiguity.
5. Write "${STATUS_PATH}" as a short plain-text summary of what was written and any remaining open questions.
6. Return JSON only with these keys:
   - `outcome`
   - `spec_path`
   - `status_path`

Set `outcome` to `ok` when the spec was written successfully.
Do not wrap the JSON in Markdown fences.
