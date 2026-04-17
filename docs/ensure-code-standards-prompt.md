Create a workflow called `ensure_code_standards` that takes no inputs.

The workflow inspects the current project (via projectdir()) and ensures that
`docs/code-standards.md` exists and accurately reflects the project's actual
languages, patterns, and conventions.

Stage 1 — analyze (new): scan source files under projectdir() to identify
languages in use (e.g. Go, Starlark, Markdown), sample representative files to
observe naming conventions, package structure, error handling, and test layout,
and check whether docs/code-standards.md already exists. Write a short analysis
artifact. Return result key `action`: one of `create` (file missing), `update`
(file exists but does not reflect what was observed in the code), or `ok` (file
is already accurate).

Stage 2 — write (new, runs only when action is `create` or `update`): using the
analysis artifact, write or rewrite docs/code-standards.md. Standards must cover
the actual languages and patterns found, follow best practices for those
languages, and be written as concrete actionable rules. The current code does not
need to comply — the file documents the target standard, not current state. Write
the file in place under projectdir().

Final outputs: artifact `standards` pointing to docs/code-standards.md, result
`action` from Stage 1, result `outcome` = `complete`.
