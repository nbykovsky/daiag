# CLI Test Report

**Date:** 2026-04-12  
**Binary:** `daiag` built from `cmd/daiag`  
**Scope:** `validate` and `run` commands — argument validation, error handling,
synthetic no-step execution, and a real Claude workflow run end-to-end

## Summary

| Result | Count |
|---|---|
| PASS | 30 |
| FAIL | 0 |
| Total | 30 |

## Workflows Executed

### Synthetic (no executor — error and validation cases)

| ID | Source | Definition |
|---|---|---|
| `simple` | test fixture | `wf = workflow(id = "simple", steps = [])` |
| `needs_input` | test fixture | declares `inputs = ["name"]`, no steps |
| `parent` / `child` | test fixtures | parent subworkflows into child, both empty |
| `bad_syntax` | test fixture | invalid Starlark (missing comma) |

### Real Claude workflow

| Workflow | File | Executor | Inputs |
|---|---|---|---|
| `spec-refinement` | `examples/development-workflow/workflows/spec-refinement/spec-refinement.star` | `claude sonnet` | `feature_dir`, `prd_path`, `spec_path` for the `indicators` feature |

The `spec-refinement` workflow ran end-to-end against
`examples/development-workflow/docs/features/indicators/prd.md`:

```
[10:24:00] workflow start id=spec-refinement
[10:24:00] step start id=write_spec           cli=claude model=sonnet
[10:25:35] step done  id=write_spec           artifacts=spec,spec_write_status  outcome=ok
[10:25:35] loop iter  id=refine_spec n=1
[10:25:35] step start id=review_spec          cli=claude model=sonnet
[10:26:24] step done  id=review_spec          artifacts=review_report,spec      outcome=not_ready
[10:26:24] step start id=address_review       cli=claude model=sonnet
[10:26:49] step done  id=address_review       artifacts=spec,spec_refine_status
[10:26:49] loop check id=refine_spec          result=stop
[10:26:49] workflow done id=spec-refinement   status=success
```

3 Claude tasks executed (write_spec → review_spec → address_review).
The reviewer returned `not_ready`, address_review returned `loop_outcome=stop`,
and the `repeat_until` loop exited after one iteration.
Total wall time: ~2m 49s.

**Artifacts written:**

| File | Description |
|---|---|
| `examples/development-workflow/docs/features/indicators/spec.md` | Technical spec produced by write_spec, refined by address_review |
| `examples/development-workflow/docs/features/indicators/spec_write_status.md` | Status summary from write_spec |
| `examples/development-workflow/docs/features/indicators/spec_review_1.md` | Review report from review_spec iteration 1 |
| `examples/development-workflow/docs/features/indicators/spec_refine_status_1.md` | Refinement status from address_review iteration 1 |

## Test Cases

### `daiag validate`

| # | Test | Exit | Checked in | Result |
|---|---|---|---|---|
| 1 | No-input synthetic workflow is valid | 0 | stdout: `workflow "simple" is valid` | PASS |
| 2 | Parent-child subworkflow is valid | 0 | stdout: `workflow "parent" is valid` | PASS |
| 3 | Real `spec-refinement` reports missing required inputs (validate has no `--input`) | 1 | stderr: `missing workflow input` | PASS |
| 4 | Real `feature-development` reports missing required inputs | 1 | stderr: `missing workflow input` | PASS |
| 5 | Missing `--workflow` flag | 2 | stderr: `--workflow is required` | PASS |
| 6 | Synthetic workflow with declared input fails without inputs | 1 | stderr: `missing workflow input "name"` | PASS |
| 7 | Unknown workflow ID | 1 | stderr: `unknown` | PASS |
| 8 | Path-style workflow ID rejected (`./simple.star`) | 1 | stderr: `workflow ID` | PASS |
| 9 | Syntax error in workflow file | 1 | stderr: `load workflow` | PASS |
| 10 | Nonexistent `--workflows-lib` | 1 | stderr: `--workflows-lib` | PASS |
| 11 | Unexpected positional argument | 2 | stderr: `unexpected arguments` | PASS |

### `daiag run` — argument errors

| # | Test | Exit | Checked in | Result |
|---|---|---|---|---|
| 12 | Missing `--workflow` flag | 2 | stderr: `--workflow is required` | PASS |
| 13 | Missing `--workdir` flag | 2 | stderr: `--workdir is required` | PASS |
| 14 | Conflicting `--input` and `--param` for same key | 2 | stderr: `conflicting workflow input` | PASS |
| 15 | Invalid `--param` format (no `=`) | 2 | stderr: `invalid --param` | PASS |
| 16 | Invalid `--input` format (no `=`) | 2 | stderr: `invalid --input` | PASS |
| 17 | Nonexistent `--workflows-lib` | 1 | stderr: `--workflows-lib` | PASS |
| 18 | Unknown workflow ID | 1 | stderr: `unknown` | PASS |
| 19 | Path-style workflow ID rejected | 1 | stderr: `workflow ID` | PASS |
| 20 | Missing input for workflow that declares it | 1 | stderr: `missing workflow input` | PASS |

### `daiag run` — synthetic execution (no executor)

| # | Test | Exit | Checked in | Result |
|---|---|---|---|---|
| 21 | No-step workflow completes | 0 | stdout: `workflow done id=simple status=success` | PASS |
| 22 | Workflow with input succeeds when input provided | 0 | stdout: `workflow done id=needs_input status=success` | PASS |
| 23 | Parent-child subworkflow executes | 0 | stdout: `workflow done id=parent status=success` | PASS |
| 24 | `--param` accepted as input alias | 0 | stdout: `workflow done id=needs_input status=success` | PASS |
| 25 | `--workdir` created if it does not exist | 0 | stdout: `workflow done id=simple status=success` | PASS |

### `daiag run` — real Claude execution

| # | Test | Exit | Checked in | Result |
|---|---|---|---|---|
| 26 | `spec-refinement` workflow with `indicators` PRD | 0 | stdout: `workflow done id=spec-refinement status=success` | PASS |

### `daiag help`

| # | Test | Exit | Checked in | Result |
|---|---|---|---|---|
| 27 | `daiag help` prints usage | 0 | stdout: `validate` | PASS |
| 28 | `daiag -h` prints usage | 0 | stdout: `validate` | PASS |
| 29 | `daiag --help` prints usage | 0 | stdout: `validate` | PASS |
| 30 | Unknown command exits 2 | 2 | stderr: `unknown command` | PASS |

## Notes

- `validate` has no `--input` flag by design — it checks workflow structure
  only. Real workflows that declare required inputs will correctly fail
  validation with `missing workflow input` when run without inputs (tests 3–4).
- Exit code 2 is reserved for argument errors; exit code 1 for load or
  execution errors. All tests confirm this distinction holds.
- The `spec-refinement` run used `--workdir /Users/nik/Projects/daiag` (the
  project root) so that relative artifact paths in the prompt templates resolve
  correctly. Claude ran with `--permission-mode bypassPermissions` and
  `--add-dir` pointing to the project root.
