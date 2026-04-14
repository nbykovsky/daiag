# CLI Test Report — workflow-bootstrap-cli.md Implementation

**Date:** 2026-04-14  
**Binary:** `daiag` (built from `cmd/daiag`)  
**Model used for live runs:** `claude-haiku-4-5-20251001`

---

## Summary

All phases passed. The implementation matches the spec in
`docs/workflow-bootstrap-cli.md` across unit tests, argument validation, static
validation, live workflow execution, and end-to-end bootstrap.

| Phase | Result |
|---|---|
| Unit tests | PASS |
| Argument validation | PASS |
| `daiag validate` | PASS |
| `daiag run` | PASS |
| `daiag bootstrap` | PASS |
| Path model edge cases | PASS |

---

## Phase 1 — Unit Tests

```
go test ./... -count=1
go vet ./...
```

| Package | Result |
|---|---|
| `internal/cli` | ok |
| `internal/executor/claude` | ok |
| `internal/executor/codex` | ok |
| `internal/logging` | ok |
| `internal/runtime` | ok |
| `internal/starlarkdsl` | ok |

`go vet` produced no output.

---

## Phase 2 — Argument Validation

All cases produce exit code 2 with the error on stderr followed by the usage
summary. Errors are printed to stderr; usage follows on stderr.

| Command | Error message | Exit |
|---|---|---|
| `daiag` (no args) | usage printed | 2 |
| `daiag run` | `--workflow is required` | 2 |
| `daiag run --workflow wf extra` | `unexpected arguments: extra` | 2 |
| `daiag validate` | `--workflow is required` | 2 |
| `daiag bootstrap` | `exactly one of --description or --description-file is required` | 2 |
| `daiag bootstrap --description a --description-file b` | `exactly one of --description or --description-file is required` | 2 |
| `daiag bootstrap --workflow "" --description a` | `--workflow must not be empty` | 2 |
| `daiag unknown` | `unknown command "unknown"` | 2 |

Deprecated flags (`--workdir`, `--param`) are correctly rejected as unknown
flags (the standard `flag` package rejects them with exit 2).

---

## Phase 3 — `daiag validate`

### All catalog workflows validate cleanly (exit 0)

```
daiag validate --workflow poem_generator
daiag validate --workflow file_row_grower
daiag validate --workflow workflow_bootstrapper
daiag validate --workflow workflow_composer
daiag validate --workflow workflow_author_from_blueprint
```

### Explicit path flags resolve correctly

```
daiag validate \
  --workflow poem_generator \
  --projectdir /Users/nik/Projects/daiag \
  --workflows-lib .daiag/workflows
```

`--workflows-lib` is relative, resolved from `--projectdir`. Exit 0.

### `--input` accepted for static validation

```
daiag validate --workflow poem_generator --input n=5
```

Exit 0. Declared `input(...)` values do not need concrete values for
validation; the flag is accepted without error.

### Error cases

| Command | Error | Exit |
|---|---|---|
| `daiag validate --workflow nonexistent_workflow` | `workflow "nonexistent_workflow" not found: expected …/nonexistent_workflow/workflow.star` | 1 |
| `daiag validate --workflow poem_generator --workflows-lib /nonexistent/path` | `--workflows-lib "/nonexistent/path": stat …: no such file or directory` | 1 |

---

## Phase 4 — `daiag run` with Claude Haiku

A test workflow `hello_claude` was created under `.daiag/workflows/hello_claude/`:

- `workflow.star` — single task `write_poem`, executor `claude-haiku-4-5-20251001`
- `write_poem.md` — prompt template with `${TOPIC}` and `${POEM_PATH}`

### Basic run

```
daiag run --workflow hello_claude --input topic=rain
```

```
[08:28:03] workflow start id=hello_claude file=…/hello_claude/workflow.star
[08:28:03] step start id=write_poem cli=claude model=claude-haiku-4-5-20251001
[08:28:14] step done id=write_poem artifacts=poem
[08:28:14] workflow done id=hello_claude status=success
workflow outputs:
artifact poem: …/.daiag/runs/hello_claude/20260414-042803-241546000Z/hello_claude/poem.txt
result poem_path: "…/hello_claude/poem.txt"
```

Exit 0. Artifact verified:

```
Rain falls softly from the sky,
Puddles form and clouds drift by,
Earth drinks deep with grateful heart,
As life begins a brand new start.
```

- Run dir created at `.daiag/runs/hello_claude/<timestamp>/` ✓
- Artifact at `<run-dir>/hello_claude/poem.txt` (under `run-dir`, not project root) ✓
- `workflow outputs:` block with `artifact` and `result` lines in sorted order ✓
- `--verbose` flag accepted without error ✓

### Relative `--run-dir`

```
daiag run --workflow hello_claude --input topic=sun --run-dir .daiag/runs/manual-test
```

Resolved from `projectdir`. Artifact at `.daiag/runs/manual-test/hello_claude/poem.txt`. Exit 0.

### `--run-dir` outside `projectdir` (containment check)

```
daiag run --workflow hello_claude --input topic=snow --run-dir /tmp/test-run-outside
```

```
error: --run-dir "/private/tmp/test-run-outside" must be inside --projectdir "/Users/nik/Projects/daiag"
```

Exit 1. macOS `/tmp` → `/private/tmp` symlink resolved correctly via
`filepath.EvalSymlinks`.

---

## Phase 5 — `daiag bootstrap` with Claude Haiku

A two-task bootstrap workflow `haiku_bootstrapper` was created:

- `plan` task — writes a blueprint `.md` file, returns `blueprint_path` + `outcome`
- `author` task — reads the blueprint, creates workflow files in `workflows-lib`,
  writes a summary, returns `workflow_id` + `workflow_path` + `outcome`

Both tasks use executor `claude-haiku-4-5-20251001`.

### Bootstrap run

```
daiag bootstrap \
  --workflow haiku_bootstrapper \
  --description "create a workflow that writes a greeting to a file"
```

```
[08:29:36] workflow start id=haiku_bootstrapper …
[08:29:36] step start id=plan cli=claude model=claude-haiku-4-5-20251001
[08:29:51] step done id=plan artifacts=blueprint outcome=complete
[08:29:51] step start id=author cli=claude model=claude-haiku-4-5-20251001
[08:30:28] step done id=author artifacts=summary outcome=complete
[08:30:28] workflow done id=haiku_bootstrapper status=success
bootstrap complete
workflow: write_greeting
workflow path: /Users/nik/Projects/daiag/.daiag/workflows/write_greeting/workflow.star
run dir: /Users/nik/Projects/daiag/.daiag/runs/bootstrap/20260414-042936-661310000Z
```

Exit 0.

Output format matches the spec exactly:

```
bootstrap complete
workflow: <workflow-id>
workflow path: <workflows-lib>/<workflow-id>/workflow.star
run dir: /abs/project/.daiag/runs/bootstrap/<run-id>
```

Generated files:

```
.daiag/workflows/write_greeting/workflow.star
.daiag/workflows/write_greeting/generate_greeting.md
.daiag/runs/bootstrap/20260414-042936-661310000Z/haiku_bootstrapper/blueprint.md
.daiag/runs/bootstrap/20260414-042936-661310000Z/haiku_bootstrapper/summary.md
```

Run artifacts are under `.daiag/runs/bootstrap/<run-id>/` (not the catalog). ✓  
Generated workflow files are under `.daiag/workflows/write_greeting/` (catalog). ✓

### Post-bootstrap validation

```
daiag validate --workflow write_greeting
```

Exit 0. The CLI validates the generated workflow automatically after bootstrap
and `daiag validate` confirms it independently.

### Post-bootstrap run of generated workflow

```
daiag run --workflow write_greeting --input greeting_name=Alice
```

```
workflow outputs:
artifact greeting_file: …/write_greeting/20260414-043044-919833000Z/write_greeting/greeting.txt
result character_count: 151
result message: "Hello Alice! We're absolutely delighted to have you here…"
```

Exit 0. The generated workflow is immediately runnable.

### `--description-file`

```
echo "create a workflow that counts words in a string" > /tmp/test-desc.txt
daiag bootstrap --workflow haiku_bootstrapper --description-file /tmp/test-desc.txt
```

Generated workflow: `count_words`. Exit 0.

### `--workflows-lib` outside `projectdir` rejected

```
daiag bootstrap --description "foo" --workflows-lib /tmp/some-dir
```

```
error: --workflows-lib "/private/var/…/tmp.XAddTWEaFf" must be inside --projectdir "/Users/nik/Projects/daiag" for bootstrap
```

Exit 1. ✓

---

## Phase 6 — Path Model

| Scenario | Behaviour | Exit |
|---|---|---|
| Default `projectdir` (nearest ancestor with `.daiag/`) | Resolved correctly from CWD | 0 |
| Explicit `--projectdir` (absolute) | Used as-is after stat | 0 |
| Default `run-dir` for `run` | `.daiag/runs/<workflow-id>/<timestamp>` | 0 |
| Default `run-dir` for `bootstrap` | `.daiag/runs/bootstrap/<timestamp>` | 0 |
| Relative `--run-dir` | Resolved from `projectdir` | 0 |
| `--run-dir` outside `projectdir` | Containment error | 1 |
| Default `workflows-lib` | `<projectdir>/.daiag/workflows` | 0 |
| Relative `--workflows-lib` | Resolved from `projectdir` | 0 |
| `--workflows-lib` outside `projectdir` (bootstrap) | Containment error | 1 |
| Nonexistent `--workflows-lib` | stat error | 1 |

Timestamp format `YYYYMMDD-HHMMSS-NNNNNNNNNZ` confirmed in run dir names:
`20260414-042803-241546000Z`.

---

## Issues Found

None. All spec requirements pass.

---

## Test Artifacts Created

The following files were added to the catalog as part of testing and can be
removed or kept:

```
.daiag/workflows/hello_claude/          # run test workflow (claude haiku)
.daiag/workflows/haiku_bootstrapper/    # bootstrap test workflow (claude haiku)
.daiag/workflows/write_greeting/        # generated by first bootstrap run
.daiag/workflows/count_words/           # generated by --description-file run
```

Run artifacts are under `.daiag/runs/` (gitignored).
