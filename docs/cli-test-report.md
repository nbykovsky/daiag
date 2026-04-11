# CLI Test Report

**Date:** 2026-04-11
**Model:** claude-haiku-4-5-20251001
**Binary:** `./daiag` (built from `cmd/daiag`)
**Workflows library:** `.daiag/workflows/`

---

## Test Workflows Created

Three workflows were written for this test run. All use the Claude executor.

### `greet`
Single task, no inputs. Writes a three-sentence greeting to `<workdir>/greeting.md`.
Returns `greeting_path` and `word_count`.

### `summarize`
Single task, one input (`topic`). Writes a five-sentence summary to `<workdir>/summary.md`.
Returns `summary_path` and `sentence_count`.

### `count_words`
Single task, one input (`sentence`). Counts words and characters, writes a report to `<workdir>/result.md`.
Returns `word_count`, `char_count`, and `result_path`.

---

## Results

### Execution tests

| # | Command | Expected | Actual | Pass |
|---|---------|----------|--------|------|
| 1 | `run --workflow greet --workdir /tmp/daiag-test-greet` | exit 0, artifact written | exit 0, `greeting.md` written | ✓ |
| 2 | `run --workflow summarize --workdir ... --input topic="Go programming language"` | exit 0, artifact written | exit 0, `summary.md` written | ✓ |
| 3 | `run --workflow count_words --workdir ... --input "sentence=The quick brown fox jumps over the lazy dog"` | exit 0, artifact written, `word_count=9` | exit 0, `result.md` written, JSON correct | ✓ |
| 9 | `run --workflow greet --workflows-lib .daiag/workflows --workdir ...` | exit 0 (explicit lib) | exit 0 | ✓ |

### Validation and error tests

| # | Command | Expected exit | Expected error | Actual | Pass |
|---|---------|---------------|----------------|--------|------|
| 4 | `run --workdir /tmp/x` (no `--workflow`) | 2 | `--workflow is required` | exit 2, correct error + usage | ✓ |
| 5 | `run --workflow greet` (no `--workdir`) | 2 | `--workdir is required` | exit 2, correct error + usage | ✓ |
| 6 | `run --workflow greet --workdir relative/path` | 1 | `--workdir must be an absolute path` | exit 1, correct error | ✓ |
| 7 | `run --workflow ./greet/greet.star --workdir /tmp/x` | 1 | path-style rejected | exit 1, `must be a workflow ID matching [A-Za-z0-9_-]+` | ✓ |
| 8 | `run --workflow does_not_exist --workdir /tmp/x` | 1 | missing file listed | exit 1, expected path shown | ✓ |
| 10 | `run --workflow greet --workflows-lib /nonexistent/path --workdir /tmp/x` | 1 | missing library error | exit 1, stat error with path | ✓ |
| 11 | `run --workflow summarize --workdir /tmp/x --input topic=foo --param topic=bar` | 2 | conflict error | exit 2, `conflicting workflow input "topic"` | ✓ |
| 12 | `help` | 0 | usage to stdout | exit 0, usage printed | ✓ |
| 13 | `validate` (unknown command) | 2 | unknown command error | exit 2, `unknown command "validate"` | ✓ |
| 14 | _(no arguments)_ | 2 | usage to stderr | exit 2, usage printed | ✓ |

**14 / 14 tests passed.**

---

## Artifact Output Samples

### Test 1 — `greet` output (`greeting.md`)

```
Hello and welcome! I'm delighted to meet you and help you with whatever you need today. Together, we'll make great things happen!
```

### Test 2 — `summarize` output (`summary.md`, topic: Go programming language)

```
Go, also known as Golang, is a compiled, statically-typed programming language created by Google in 2009, designed for simplicity and efficiency in building scalable software. The language emphasizes clean syntax and a minimalist philosophy, omitting features like inheritance and complex type hierarchies in favor of composition and interfaces that promote code clarity. Go's built-in concurrency model uses lightweight goroutines and channels, making it exceptionally well-suited for writing concurrent programs that efficiently utilize multi-core processors without the complexity of traditional threading. The language is particularly popular for building backend services, cloud infrastructure, microservices, and command-line tools, with widespread adoption by companies like Kubernetes, Docker, and Uber. Go's strong standard library, fast compilation times, cross-platform support, and built-in tooling for testing, formatting, and documentation make it an attractive choice for modern systems programming and DevOps applications.
```

### Test 3 — `count_words` output (`result.md`, input: "The quick brown fox jumps over the lazy dog")

```
Words: 9, Characters: 43
```

JSON extracted correctly: `word_count=9`, `char_count=43`. Both values are correct.

---

## Observations

**Exit codes are correct throughout.** Argument errors exit 2; execution/load errors exit 1; success exits 0.

**Usage is printed on the right stream.** Argument errors print usage to stderr (test 4, 5, 11, 13); `help` prints to stdout (test 12).

**Error messages are actionable.** The missing workflow error names the exact expected path. The path-style rejection explains the ID pattern. The `--workflows-lib` stat error includes the resolved absolute path.

**Workdir is created automatically.** None of the test workdirs were pre-created; the CLI created them before execution in all three run tests.

**`--workflows-lib` relative path resolves correctly.** `.daiag/workflows` (relative) was accepted and resolved from CWD (test 9).

**JSON result extraction works.** Claude returned plain JSON matching `result_keys` in all three workflows; the runner extracted keys without error.

**One minor inconsistency:** relative `--workdir` exits with code 1 (test 6) rather than 2. Argument validation errors are otherwise consistently code 2; this one is caught during runner setup rather than flag parsing.

---

## Issues Found

| Severity | Description |
|----------|-------------|
| Minor | `--workdir must be an absolute path` exits with code 1 instead of 2. It is an argument error and should exit 2 for consistency with other argument errors. |