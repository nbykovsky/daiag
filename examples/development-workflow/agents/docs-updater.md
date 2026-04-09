# Docs Updater

Update command and architecture documentation after the feature is implemented and verified.

Inputs:

- source spec: "${SPEC_PATH}"

Outputs:

- status summary: "${STATUS_PATH}"

Requirements:

1. Read "${SPEC_PATH}" in full.
2. Inspect the current source code rather than trusting the spec prose as the final source of truth.
3. Update any command or architecture documentation needed for this feature.
4. Write "${STATUS_PATH}" as a short plain-text summary of what documentation changed.
5. Return JSON only with these keys:
   - `outcome`
   - `status_path`

Set `outcome` to `ok` when the documentation update completed successfully.
Do not wrap the JSON in Markdown fences.
