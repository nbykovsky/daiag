You are writing or updating the code standards document for a software project.

Analysis artifact path: ${ANALYSIS_PATH}
Output path: ${STANDARDS_PATH}

## Task

1. Read the **full content** of the analysis at `${ANALYSIS_PATH}` before writing anything. It contains:
   - Detected languages and file types
   - Observed naming and structural conventions
   - Error-handling and test patterns
   - Project-specific idioms
   - The full text of any existing standards file (if present)
   - An `action` judgment: `create` or `update`

2. Check the `action` judgment in the analysis and proceed accordingly:

   - **`create`** — no existing standards file was found. Write the document from scratch based entirely on the observed codebase conventions.
   - **`update`** — an existing standards file was found and its full text is included in the analysis. Merge and improve it: preserve correct and relevant rules, add missing rules discovered in the analysis, remove rules that no longer reflect the codebase, and resolve any inconsistencies. Do not discard content without a reason.

3. Write or rewrite the file at `${STANDARDS_PATH}`.
   The document must:
   - Cover all detected languages and concern areas found in the analysis
   - Provide concrete, actionable rules — not vague guidelines
   - Be organized by language or concern area with clear headings
   - Explicitly note that existing code may not yet comply with these standards

4. Write the complete file content to `${STANDARDS_PATH}`.

Return JSON with:
- `outcome`: `complete`

Do not wrap the JSON in Markdown fences.
