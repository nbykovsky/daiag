# Workflow Description: ensure_code_standards

Create a workflow called `ensure_code_standards` with no user-provided inputs.
The workflow discovers everything it needs from the project directory at runtime.

## Goal

Ensure that `docs/code-standards.md` exists in the project root and is relevant
to the actual codebase. If the file is missing, create it. If it exists but is
stale or does not reflect the real languages, patterns, or conventions in use,
update it in place. The standards should follow best practices for the detected
stack — the current code does not need to comply with them; conformance is a
separate concern.

## Stages

### Stage 1: analyze (new stage)

Scan the project to understand what it contains:

- List source files and identify the primary languages and file types in use
  (e.g. Go, Starlark, Markdown).
- Sample a representative set of files to observe naming conventions, package
  structure, error handling patterns, test layout, and any project-specific idioms.
- Check whether `docs/code-standards.md` exists and, if so, read its current
  contents.

Inputs: project directory path (from `projectdir()`).

Outputs: a written analysis artifact containing — the detected languages and
file types, a summary of observed conventions, the full text of the existing
standards (if any), and a judgment: `create`, `update`, or `ok`.

Result key: `action` — one of `create`, `update`, `ok`.

Done when the artifact exists and `action` is one of the three values.

### Stage 2: write (new stage, runs when action is `create` or `update`)

Using the analysis artifact from Stage 1, write or rewrite `docs/code-standards.md`.

The standards must:
- Cover the languages and patterns actually present in the project.
- Follow best practices for those languages (e.g. effective Go idioms, clear
  Starlark conventions).
- Be structured as concrete, actionable rules a developer can follow.
- Not require the current codebase to comply — note that existing code may
  diverge and will be brought into conformance separately.

If the file already exists, update it in place. If it does not exist, create it.

Inputs: analysis artifact path (from Stage 1), project directory path.

Outputs: `docs/code-standards.md` updated or created under `projectdir()`.

Result key: `outcome` — `complete`.

Done when `docs/code-standards.md` exists and contains the new standards.

## Final outputs

- Artifact `analysis`: the analysis document from Stage 1.
- Artifact `standards`: `docs/code-standards.md`.
- Result `action`: the judgment from Stage 1 (`create`, `update`, or `ok`).
- Result `outcome`: `complete`.

## Notes

- Both stages use the Claude executor.
- If `action` is `ok`, Stage 2 should be skipped (use a `when` conditional).
- The workflow has no declared inputs; all paths come from `projectdir()` and
  `run_dir()` at runtime.
