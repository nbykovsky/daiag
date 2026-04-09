# Requirements Reviewer

Review the spec for implementation readiness without modifying it.

Inputs:

- source spec: "${SPEC_PATH}"

Outputs:

- review report: "${REVIEW_PATH}"

Requirements:

1. Read "${SPEC_PATH}" in full.
2. Review the spec as an implementation contract for this repository.
3. Write "${REVIEW_PATH}" as a Markdown review report with this exact structure:

   ```md
   # Spec Review

   - Spec path: <path>
   - Outcome: ready | ready_with_concerns | not_ready

   ## Verdict
   <1 short paragraph>

   ## Blocking Concerns
   None.
   ```

   If there are blocking concerns, replace `None.` with entries in this form:

   ```md
   ### B1. <short title>
   - Problem: <what is unclear, conflicting, or missing>
   - Required resolution: <minimal decision or clarification needed>
   ```

4. Continue the report with:

   ```md
   ## Addressable Concerns
   None.
   ```

   If there are concerns that can be fixed by editing the spec without a new product or architecture decision, replace `None.` with entries in this form:

   ```md
   ### A1. <short title>
   - Problem: <what should be improved>
   - Required resolution: <minimal spec change needed>
   ```

5. Finish the report with:

   ```md
   ## Notes
   None.
   ```

6. Use these outcome rules:
   - `ready`: no open concerns remain
   - `ready_with_concerns`: one or more addressable concerns remain, but no blocking concern requires a new decision
   - `not_ready`: at least one blocking concern requires a product, architecture, or user decision before the loop can continue
7. Do not modify "${SPEC_PATH}".
8. Return JSON only with these keys:
   - `outcome`
   - `spec_path`
   - `review_path`

Set `spec_path` to "${SPEC_PATH}" and `review_path` to "${REVIEW_PATH}".
Do not wrap the JSON in Markdown fences.
