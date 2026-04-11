# Plan: `daiag validate` command

## Goal

Add a `daiag validate` subcommand that loads and parses a Starlark workflow
file, reports errors, and exits without executing any tasks.

## Command surface

```sh
daiag validate --workflow <id> [--workflows-lib <dir>]
```

- `--workflow` — required, same rules as `run`
- `--workflows-lib` — optional, same defaulting logic as `run`
- No `--workdir`, `--input`, `--param` — not needed for parse-only validation

**Note:** a workflow that declares `inputs = [...]` will fail with
"missing workflow input" because validate passes an empty input map. This is
correct — `validate` checks structure, not a full dry-run with real values.
Integration tests use a no-input fixture workflow to test the success path (see
section 4 below).

## Output and error message format

### Success

Printed to stdout, exit 0:

```
workflow "parent" is valid
```

Produced by `fmt.Fprintf(a.stdout, "workflow %q is valid\n", cfg.Workflow)`.
The `%q` verb matches the quoting style used elsewhere in the codebase (e.g.
`conflicting workflow input "name"`, `--workflows-lib %q is not a directory`).

### Errors — exit 1

Errors from the loader and validator are returned as-is from `starlarkdsl`
and printed by `App.Run` as:

```
error: <message from loader/validator>
```

Concrete examples:

```
error: missing workflow input "name"
error: workflow reference "parent.star" must be a workflow ID
error: --workflows-lib "/bad/path": stat /bad/path: no such file or directory
```

This matches the existing `run` error format in `App.Run`:
`fmt.Fprintf(a.stderr, "error: %v\n", err)`.

### Argument errors — exit 2

Flag parse errors and missing required flags print the error followed by the
usage summary:

```
error: --workflow is required

Usage:
  daiag run ...
  daiag validate ...
```

Produced by the same pattern as the `run` case:
`fmt.Fprintf(a.stderr, "error: %v\n\n", err)` then `a.printUsage(a.stderr)`.

### Cascading error handling

`parseValidateArgs` returns on the first error (flag parse failure or missing
`--workflow`). These are the only argument-layer errors; `--workflows-lib` is
not validated at parse time — it surfaces from `workflowValidator.Validate` via
`resolveWorkflowsLib` with exit 1. It is not possible to trigger both a missing
`--workflow` and an invalid `--workflows-lib` simultaneously at exit-2 level
since the workflow check fails first.

## Context source

`App.Run(ctx context.Context, args []string)` receives `ctx` from the caller.
In `main.go` this is `context.Background()`. The validate branch passes it
directly to `a.validator.Validate(ctx, cfg)`, the same as the run branch passes
it to `a.runner.Run(ctx, cfg)`.

## What the codebase already provides

`starlarkdsl.Loader.Load` already parses the Starlark file and runs the full
validation pass. The validate command calls that and stops — it does not reach
into `runtime.Engine`.

`default_runner.go` already contains `resolveWorkflowsLib`,
`starlarkdsl.ResolveWorkflowID`, and `starlarkdsl.Loader.Load` in sequence.
The validator reuses the same three steps.

Confirmed signatures (from `internal/starlarkdsl`):

```go
func (l Loader) Load(path string) (*workflow.Workflow, error)
func ResolveWorkflowID(workflowsLib string, id string) (string, error)
```

`workflowValidator.Validate` discards the `*workflow.Workflow` return value —
only the error matters. `ResolveWorkflowID` returns the resolved `.star` file
path, which is passed directly to `Load`.

`App.Run` already switches on `args[0]` (confirmed in `app.go:52`). All five
existing `New(...)` call sites in `app_test.go` use the three-argument form and
need a `nil` fourth argument added.

Existing test helpers in `default_runner_test.go`:

| Helper | What it does |
|---|---|
| `writeCLITestWorkflow(t, lib)` | Creates `parent` (requires `name` input) and `child` workflows under `lib` |
| `writeCLITestFile(t, path, contents)` | Writes any file to disk, creating parent dirs |
| `withWorkingDir(t, dir)` | Changes CWD for the test and restores it on cleanup |

## Files to change

### 1. `internal/cli/app.go`

- Add `ValidateConfig` struct with `Workflow string` and `WorkflowsLib string`.
- Add `Validator` interface with `Validate(context.Context, ValidateConfig) error`.
- Add `validator Validator` field to `App`.
- Update `New` signature to `New(stdout, stderr io.Writer, runner Runner, validator Validator) *App`.
- Update `usageText` to list `validate`:
  ```
  Commands:
    run       Execute a workflow
    validate  Parse and validate a workflow without executing it
  ```
- Add `"validate"` case in `App.Run`:
  ```go
  case "validate":
      cfg, err := parseValidateArgs(args[1:])
      if err != nil {
          fmt.Fprintf(a.stderr, "error: %v\n\n", err)
          a.printUsage(a.stderr)
          return 2
      }
      if err := a.validator.Validate(ctx, cfg); err != nil {
          fmt.Fprintf(a.stderr, "error: %v\n", err)
          return 1
      }
      fmt.Fprintf(a.stdout, "workflow %q is valid\n", cfg.Workflow)
      return 0
  ```
- Add `parseValidateArgs`:
  ```go
  func parseValidateArgs(args []string) (ValidateConfig, error) {
      var cfg ValidateConfig
      fs := flag.NewFlagSet("validate", flag.ContinueOnError)
      fs.SetOutput(io.Discard)
      fs.StringVar(&cfg.Workflow, "workflow", "", "workflow ID")
      fs.StringVar(&cfg.WorkflowsLib, "workflows-lib", "", "workflow library directory")
      if err := fs.Parse(args); err != nil {
          return ValidateConfig{}, err
      }
      if fs.NArg() > 0 {
          return ValidateConfig{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
      }
      if cfg.Workflow == "" {
          return ValidateConfig{}, errors.New("--workflow is required")
      }
      return cfg, nil
  }
  ```

### 2. `internal/cli/app_test.go`

- Update all five existing `New(&stdout, &stderr, &fakeRunner{})` call sites to
  pass `nil` as the fourth argument — existing tests never exercise the validate
  path so nil is safe (matches existing nil-tolerance for `runner`).
- Add `fakeValidator`:
  ```go
  type fakeValidator struct {
      called bool
      cfg    ValidateConfig
      err    error
  }
  func (f *fakeValidator) Validate(_ context.Context, cfg ValidateConfig) error {
      f.called = true
      f.cfg = cfg
      return f.err
  }
  ```
- Add four tests using `New(&stdout, &stderr, nil, &fakeValidator{...})`:
  - `TestAppValidateMissingWorkflow` — no `--workflow` → exit 2, stderr contains `"--workflow is required"`
  - `TestAppValidateInvokesValidator` — valid flags → `fakeValidator.called == true`, `cfg.Workflow == "poem"`, exit 0
  - `TestAppValidatePropagatesValidatorError` — `fakeValidator{err: errors.New("boom")}` → exit 1, stderr contains `"boom"`
  - `TestAppValidateSuccessPrintsMessage` — success → stdout contains `workflow "poem" is valid`

### 3. `internal/cli/default_runner.go`

- Add `workflowValidator` struct:
  ```go
  type workflowValidator struct{}

  func (v workflowValidator) Validate(_ context.Context, cfg ValidateConfig) error {
      workflowsLib, err := resolveWorkflowsLib(cfg.WorkflowsLib)
      if err != nil {
          return err
      }
      workflowPath, err := starlarkdsl.ResolveWorkflowID(workflowsLib, cfg.Workflow)
      if err != nil {
          return err
      }
      loader := starlarkdsl.Loader{BaseDir: workflowsLib}
      _, err = loader.Load(workflowPath)
      return err
  }
  ```
- Update `NewDefault`:
  ```go
  func NewDefault(stdout, stderr io.Writer) *App {
      return New(stdout, stderr, workflowRunner{stdout: stdout}, workflowValidator{})
  }
  ```

### 4. `internal/cli/default_runner_test.go`

The existing `writeCLITestWorkflow` fixture declares `inputs = ["name"]`, so
it cannot be used as the success case for `validate` (which passes an empty
input map). Add a `writeValidateTestWorkflow` helper that writes a minimal
no-input workflow:

```go
func writeValidateTestWorkflow(t *testing.T, workflowsLib string) {
    t.Helper()
    writeCLITestFile(t, filepath.Join(workflowsLib, "simple", "simple.star"), `
wf = workflow(id = "simple", steps = [])
`)
}
```

Add three integration tests:

- `TestDefaultRunnerValidatesValidWorkflow` — `writeValidateTestWorkflow`, call
  `validate --workflow simple --workflows-lib ...`, expect exit 0 and stdout
  `workflow "simple" is valid`.
- `TestDefaultRunnerValidateRejectsMissingInput` — `writeCLITestWorkflow` (the
  existing parent workflow which requires `name`), call
  `validate --workflow parent --workflows-lib ...` with no `--input`, expect
  exit 1 and stderr contains `missing workflow input "name"`.
- `TestDefaultRunnerValidateRejectsUnknownWorkflow` — create an empty
  `workflowsLib` dir (no star files), call `validate --workflow unknown
  --workflows-lib ...`, expect exit 1 and stderr contains `"unknown"` (the
  error from `ResolveWorkflowID`).

### 5. `docs/workflow-cli.md`

Add `validate` to the synopsis command list and insert a full `## daiag validate`
section between `## daiag run` and `## daiag help`:

```markdown
## `daiag validate`

Loads and validates a workflow without executing any tasks.

​```sh
daiag validate --workflow <workflow-id> [--workflows-lib <dir>]
​```

### Flags

#### `--workflow <workflow-id>` (required)

The workflow ID to validate. Follows the same rules as `daiag run`.

#### `--workflows-lib <dir>` (optional)

Path to the workflows library directory. Follows the same defaulting rules as
`daiag run`.

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | Workflow is valid |
| 1 | Workflow load or validation error |
| 2 | Argument error |

### Examples

Validate a workflow from the default library:

​```sh
daiag validate --workflow write_poem
​```

Validate a workflow from an explicit library:

​```sh
daiag validate --workflow feature-development \
  --workflows-lib examples/development-workflow/workflows
​```
```

## Implementation order

1. `app.go` — define `ValidateConfig`, `Validator`, update `App`, `New`,
   `usageText`, `App.Run`, add `parseValidateArgs`
2. `app_test.go` — update `New(...)` call sites, add `fakeValidator` and four
   unit tests
3. `default_runner.go` — add `workflowValidator`, update `NewDefault`
4. `default_runner_test.go` — add `writeValidateTestWorkflow` helper and three
   integration tests
5. `docs/workflow-cli.md` — update synopsis and add the validate section

## Design notes

- The `Validator` interface is justified immediately: the real implementation
  and the test fake are two concrete uses on day one, satisfying the CLAUDE.md
  two-use rule.
- `workflowValidator.Validate` has no filesystem side effects — it deliberately
  skips `resolveWorkdir`.
- `context.Context` is included on the interface method for consistency with
  `Runner`; it is unused in v1.
- Passing `nil` for `validator` in tests that only exercise `run` matches the
  existing nil-tolerance pattern — `App` does not nil-check `runner` either.
