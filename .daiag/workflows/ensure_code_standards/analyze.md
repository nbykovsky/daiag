You are analyzing a software project to document its code standards.

Project directory: ${PROJECT_DIR}
Analysis output path: ${ANALYSIS_PATH}

## Task

1. Scan the project at `${PROJECT_DIR}`. Examine:
   - Languages and file types present
   - Naming conventions (files, packages, functions, variables)
   - Package and module structure
   - Error-handling patterns
   - Test layout and conventions
   - Any project-specific idioms (e.g. CLAUDE.md, README, Makefile conventions)

2. Check whether `${PROJECT_DIR}/docs/code-standards.md` exists.
   - If it exists, read it and include its full text in the analysis artifact.
   - If it does not exist, note that it is absent.

3. Determine `action`:
   - `ok` — the standards file exists and accurately reflects the codebase as observed
   - `update` — the standards file exists but is outdated, incomplete, or inconsistent with what you observed
   - `create` — the standards file does not exist

4. Write a structured analysis to `${ANALYSIS_PATH}` covering:
   - Detected languages and file types
   - Observed naming conventions
   - Package and module structure
   - Error-handling patterns
   - Test conventions
   - Project-specific idioms
   - Full text of the existing standards file (if present)
   - Your `action` judgment with brief reasoning

Return JSON with:
- `action`: one of `create`, `update`, `ok`

Do not wrap the JSON in Markdown fences.
