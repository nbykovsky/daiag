# Task: Review Workflow

Inputs:
- `workflow_id`: ${WORKFLOW_ID}
- `workflows_dir`: ${WORKFLOWS_DIR}
- `review_path`: ${REVIEW_PATH}

Instructions:

1. Read the file at `${WORKFLOWS_DIR}/workflow.star`. This is the Starlark workflow definition to review.

2. List all `.md` files in `${WORKFLOWS_DIR}/` (excluding any subdirectories). Read each one — these are the prompt templates for the workflow's tasks.

3. Analyze the workflow definition and prompt templates against the following DSL best practices:

   **Workflow structure**
   - `workflow_id` is declared as a top-level string variable and matches the directory name
   - All inputs are declared via `input(...)` and passed through `vars` or used to construct paths
   - `run_dir()` is used for all run artifact paths; `projectdir()` is used only for project-level source files
   - Every artifact value uses `artifact(...)`
   - `output_artifacts` uses `path_ref(...)` and `output_results` uses `json_ref(...)`
   - Both `output_artifacts` and `output_results` are declared (use `{}` when empty)
   - `default_executor` specifies both `cli` and `model`
   - Steps use `template_file(...)` for prompts, not inline strings

   **Task design**
   - Each task has a focused, single responsibility
   - `result_keys` lists every key the prompt instructs the agent to return
   - Artifact keys in `artifacts` match the keys referenced by downstream `path_ref` calls
   - Downstream task inputs from prior tasks use `path_ref` or `json_ref` — not re-derived paths
   - Step decomposition is appropriate: neither too coarse (one task doing unrelated things) nor too fine (unnecessary splits)

   **Prompt template quality**
   - Each prompt names exactly which files to read and write
   - Placeholders match `vars` keys in `workflow.star`
   - Every JSON result key listed in the prompt appears in the task's `result_keys`
   - Allowed values for enum fields (e.g. `outcome`) are listed explicitly
   - The prompt ends with `Do not wrap the JSON in Markdown fences.`
   - Instructions are unambiguous and actionable for an AI agent
   - Inputs section lists all injected variables with their resolved values

   **Input and output design**
   - Workflow inputs are the minimal set needed; nothing is hard-coded that should be an input
   - Output artifacts cover the primary deliverables a caller would need
   - Output results expose the key scalar values a caller or human would inspect

4. Write the review document to `${REVIEW_PATH}`. Structure it as follows:

   ```
   # Workflow Review: <workflow_id>

   ## Summary
   <2–4 sentence overall assessment>

   ## Findings

   ### <Finding Title> [ISSUE|SUGGESTION|PRAISE]
   **Location**: <file and line or section>
   **Observation**: <what was found>
   **Recommendation**: <concrete fix or action, or "None required" for PRAISE>

   (repeat for each finding)

   ## Verdict
   <overall verdict: APPROVED | APPROVED_WITH_SUGGESTIONS | NEEDS_REVISION>
   <one sentence rationale>
   ```

   Include at least one finding per evaluated category. Label each finding as ISSUE (must fix), SUGGESTION (improvement), or PRAISE (done well).

5. Return JSON with:
   - `review_path`: the absolute path `${REVIEW_PATH}`
   - `outcome`: `"complete"`

Do not wrap the JSON in Markdown fences.
