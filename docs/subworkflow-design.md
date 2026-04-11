# Subworkflow Design

## Purpose

This document specifies planned subworkflow support for `daiag`.

Subworkflows let one workflow compose other workflows as reusable components.
The goal is to make larger agent processes easier to build without flattening
every step into one large entry workflow file.

This is an implementation design, not current behavior.

## Problem

Today a workflow is one flat execution graph made of:

- `task(...)`
- `repeat_until(...)`

Starlark `load(...)` allows authoring reuse, but loaded modules are helper
modules. They must not define top-level `wf`, and they do not create a runtime
component boundary.

That works for task helpers and small reusable step groups, but it is weak for
larger processes such as:

- spec refinement
- task planning
- implementation
- code review
- QA refinement
- documentation updates

Those stages often want to be workflows in their own right, with their own
internal task IDs, loops, artifacts, and validation rules.

Without subworkflows, authors have two bad options:

- flatten all stages into a single workflow and manually avoid step ID
  collisions
- shell out to `daiag run --workflow ...` from a task and lose structured
  references, validation, and output wiring

## Goals

- Allow a workflow to run another workflow as a first-class step.
- Give each subworkflow its own internal step ID scope.
- Require reusable workflows to declare their public inputs.
- Require reusable workflows to declare their public outputs.
- Let parent workflows pass literal values, formatted values, `path_ref(...)`,
  and `json_ref(...)` values into a subworkflow.
- Let parent workflows consume only the subworkflow's declared public outputs.
- Preserve sequential execution for v1.
- Keep the implementation small and explicit.

## Non-Goals

This feature does not add:

- parallel subworkflow execution
- dynamic fan-out over generated task files
- branching beyond existing `repeat_until(...)`
- workflow resume
- remote workflow loading
- versioned workflow packages
- task-level `inputs = ...` fields
- rich type checking for workflow inputs

## Design Summary

Add three workflow-level contract fields:

- `inputs`
- `output_artifacts`
- `output_results`

Add two DSL builtins:

- `input(name)`
- `subworkflow(...)`

From outside, a subworkflow behaves like one workflow step:

- `path_ref("spec", "spec")` reads the subworkflow artifact output named
  `spec`
- `json_ref("spec", "outcome")` reads the subworkflow result output named
  `outcome`

From inside, a subworkflow can only read:

- its declared `input(...)` values
- its own earlier task outputs
- its own loop state

This prevents child workflows from reaching into parent internals and prevents
parent workflows from reaching into child internals.

## Workflow Inputs

### Syntax

```python
draft_spec = input("spec_path")
requirements_doc = input("prd_path")

wf = workflow(
    id = "spec_refinement",
    inputs = ["spec_path", "prd_path"],
    steps = [
        # ...
    ],
)
```

### Rules

- `inputs` is optional for compatibility.
- Every declared input name must be a non-empty unique string.
- `input(name)` must refer to a declared workflow input.
- The local Starlark variable name may differ from the workflow input name.
- Top-level workflow `inputs` define the values accepted from the CLI.
- Subworkflows receive input values from their parent `subworkflow(inputs = ...)`
  map.
- `input(...)` produces a runtime value expression.
- For v1, input values are untyped runtime values.
- When an input is used in a string expression context, it is stringified with
  the same behavior used by `format(...)`.

### Inputs Replace `param(...)`

`param(...)` is the current load-time CLI parameter mechanism.
It should become compatibility-only.

New workflows should use `workflow(inputs = [...])` and `input(...)`.
The same declaration supports both top-level workflows and subworkflows:

- CLI input bindings for a top-level run
- parent workflow expressions for a subworkflow run

This means workflow authors do not need separate `param(...)` calls.

Recommended new style:

```python
feature_name = input("name")

wf = workflow(
    id = "feature_development",
    inputs = ["name"],
    steps = [],
)
```

Compatibility behavior:

- Existing `param(...)` workflows continue to load and run.
- `input(...)` workflows require the runner to provide declared workflow inputs.
- A top-level workflow may use both `param(...)` and `input(...)` only during
  migration.
- New workflows should not use `param(...)`.
- A workflow loaded through `subworkflow(...)` must not call `param(...)`.

Coexistence detail:

- `param(...)` resolves during Starlark evaluation and returns a plain string.
- `input(...)` resolves at runtime and returns a symbolic value expression.
- After Starlark evaluation, the validator cannot reliably prove that a workflow
  used both for the same logical value.
- During migration, the CLI should merge `--input` and compatibility `--param`
  values into one input map and reject conflicting values for the same key.
- If a top-level workflow uses both `param("name")` and `input("name")`, both
  should receive the same merged value.

Authoring recommendation:

- Prefer declaring `input(...)` values near the top of the workflow file that
  owns the `wf` value.
- Helper modules should prefer accepting input-derived values as function
  arguments instead of calling `input(...)` internally.

Reason:

- A hidden `input(...)` inside a helper module makes the reusable workflow's
  contract harder to see from the workflow entry file.

Subworkflow parameter rule:

- `param(...)` should be disabled while loading a workflow referenced by
  `subworkflow(...)` and any helper modules it loads.
- If a child workflow needs a value, declare it in `workflow(inputs = [...])` and
  read it with `input(...)`.
- This keeps the subworkflow boundary explicit and prevents child workflows from
  silently depending on top-level CLI input bindings.

## Workflow Outputs

### Syntax

```python
wf = workflow(
    id = "spec_refinement",
    inputs = ["feature_dir", "prd_path", "spec_path"],
    steps = [
        # write_spec
        # repeat_until review/address loop
    ],
    output_artifacts = {
        "spec": path_ref("address_review", "spec"),
        "last_review": path_ref("review_spec", "review_report"),
    },
    output_results = {
        "outcome": json_ref("address_review", "loop_outcome"),
    },
)
```

### Rules

- `output_artifacts` is optional.
- `output_results` is optional.
- Output keys must be non-empty unique strings within their own map.
- `output_artifacts` values must be string expressions.
- `output_results` values must be value expressions.
- Output expressions may reference only steps visible inside the workflow.
- Output expressions may reference declared workflow inputs. This supports
  pass-through outputs such as `output_artifacts = {"spec": input("spec_path")}`.
- Output expressions may reference loop body tasks if those tasks have executed
  before the workflow completes.
- Parent workflows can reference only declared outputs.

### Output Kinds

Artifact outputs become parent-visible artifact keys:

```python
path_ref("spec", "spec")
```

Result outputs become parent-visible result keys:

```python
json_ref("spec", "outcome")
```

The parent does not see the child's internal task IDs such as `write_spec`,
`review_spec`, or `address_review`.

## Subworkflows In Loops

A subworkflow may appear inside `repeat_until(...)`.

Rules:

- The subworkflow node ID is still unique within the parent workflow scope.
- The subworkflow may receive `loop_iter(...)` values from the parent loop when
  the subworkflow node is structurally inside that loop.
- Parent references to the subworkflow use the latest successful subworkflow
  execution, matching existing task reference behavior inside loops.
- Child workflow loop state remains isolated from parent workflow loop state.

## Subworkflow Node

### Syntax

```python
feature_name = input("name")
feature_workspace = format("docs/features/{name}", name = feature_name)

wf = workflow(
    id = "feature_development",
    inputs = ["name"],
    steps = [
        subworkflow(
            id = "spec",
            workflow = "spec-refinement.star",
            inputs = {
                "feature_dir": feature_workspace,
                "prd_path": format("{dir}/prd.md", dir = feature_workspace),
                "spec_path": format("{dir}/spec.md", dir = feature_workspace),
            },
        ),
        task(
            id = "write_qa_tests",
            prompt = template_file(
                "../agents/qa-test-writer.md",
                vars = {
                    "SPEC_PATH": path_ref("spec", "spec"),
                    "QA_TESTS_PATH": "docs/features/example/qa_tests.md",
                    "STATUS_PATH": "docs/features/example/qa_status.md",
                },
            ),
            artifacts = {
                "qa_tests": artifact("docs/features/example/qa_tests.md"),
                "status": artifact("docs/features/example/qa_status.md"),
            },
            result_keys = ["outcome", "qa_tests_path", "status_path"],
        ),
    ],
)
```

### Required Fields

- `id`
- `workflow`

### Optional Fields

- `inputs`

### Rules

- `id` must be non-empty.
- `workflow` must be a local `.star` path.
- `workflow` path resolution should match `load(...)` path resolution:
  - relative to the Starlark module where `subworkflow(...)` appears
  - under the workflow base directory
  - no URLs
  - no path escape through `..`
- `inputs` must be a dict when provided.
- omitted `inputs` defaults to `{}`.
- `inputs = {}` is valid when the child workflow declares no inputs.
- Every key in `inputs` must match a declared input on the child workflow.
- Every child required input must be provided by the parent.
- Parent input values may be:
  - string literals
  - integers
  - `input(...)`
  - `format(...)`
  - `path_ref(...)`
  - `json_ref(...)`
  - `loop_iter(...)` when structurally inside the active loop
- The parent may reference the subworkflow only after the subworkflow node has
  executed.
- The subworkflow node's public artifact and result keys come from the child
  workflow's output declarations.

## Example

Child workflow:

```python
# examples/development-workflow/workflows/spec-refinement.star

feature_workspace = input("feature_dir")
requirements_doc = input("prd_path")
draft_spec = input("spec_path")

wf = workflow(
    id = "spec_refinement",
    inputs = ["feature_dir", "prd_path", "spec_path"],
    default_executor = {"cli": "claude", "model": "sonnet"},
    steps = [
        task(
            id = "write_spec",
            prompt = template_file(
                "../agents/spec-writer.md",
                vars = {
                    "FEATURE_DIR": feature_workspace,
                    "PRD_PATH": requirements_doc,
                    "SPEC_PATH": draft_spec,
                    "STATUS_PATH": format("{dir}/spec_write_status.md", dir = feature_workspace),
                },
            ),
            artifacts = {
                "spec": artifact(draft_spec),
                "status": artifact(format("{dir}/spec_write_status.md", dir = feature_workspace)),
            },
            result_keys = ["outcome", "spec_path", "status_path"],
        ),
        repeat_until(
            id = "refine_spec",
            max_iters = 3,
            steps = [
                # review_spec
                # address_review
            ],
            until = eq(json_ref("address_review", "loop_outcome"), "stop"),
        ),
    ],
    output_artifacts = {
        "spec": path_ref("address_review", "spec"),
    },
    output_results = {
        "outcome": json_ref("address_review", "loop_outcome"),
    },
)
```

Parent workflow:

```python
# examples/development-workflow/workflows/feature-development.star

feature_name = input("name")
feature_workspace = format("examples/development-workflow/docs/features/{name}", name = feature_name)

wf = workflow(
    id = "feature_development",
    inputs = ["name"],
    steps = [
        subworkflow(
            id = "spec",
            workflow = "spec-refinement.star",
            inputs = {
                "feature_dir": feature_workspace,
                "prd_path": format("{dir}/prd.md", dir = feature_workspace),
                "spec_path": format("{dir}/spec.md", dir = feature_workspace),
            },
        ),
        task(
            id = "write_qa_tests",
            prompt = template_file(
                "../agents/qa-test-writer.md",
                vars = {
                    "SPEC_PATH": path_ref("spec", "spec"),
                    "QA_TESTS_PATH": format("{dir}/qa_tests.md", dir = feature_workspace),
                    "STATUS_PATH": format("{dir}/qa_test_write_status.md", dir = feature_workspace),
                },
            ),
            artifacts = {
                "qa_tests": artifact(format("{dir}/qa_tests.md", dir = feature_workspace)),
                "status": artifact(format("{dir}/qa_test_write_status.md", dir = feature_workspace)),
            },
            result_keys = ["outcome", "qa_tests_path", "status_path"],
        ),
    ],
)
```

## Data Model Changes

Add workflow inputs and outputs:

```go
type Workflow struct {
    ID              string
    Inputs          []string
    DefaultExecutor *ExecutorConfig
    Steps           []Node
    OutputArtifacts map[string]StringExpr
    OutputResults   map[string]ValueExpr
}
```

Add a subworkflow node:

```go
type Subworkflow struct {
    ID           string
    WorkflowPath string
    ModuleDir    string
    Workflow     *Workflow
    Inputs       map[string]ValueExpr
}

func (*Subworkflow) node() {}

func (s *Subworkflow) NodeID() string {
    return s.ID
}
```

`Workflow` is populated by the Starlark loader before runtime execution.
The child workflow's `OutputArtifacts` and `OutputResults` fields are the
single source of truth for the subworkflow's public outputs.

`ModuleDir` is populated by the `subworkflow(...)` builtin from the current
caller module path, using the same call-stack approach as `template_file(...)`.
This makes the `workflow` path stable when `subworkflow(...)` appears in a
loaded helper module.

Add workflow input references:

```go
type InputRef struct {
    Name string
}

func (InputRef) valueExpr() {}
func (InputRef) stringExpr() {}
```

`InputRef` should be accepted anywhere a string expression or value expression
is currently accepted.

Required closed-switch updates:

- `internal/starlarkdsl.unpackSteps` must accept `subworkflow(...)` values.
- `internal/starlarkdsl.unpackStringExpr` must accept `input(...)` values.
- `internal/starlarkdsl.unpackValueExpr` must accept `input(...)` values.
- `internal/workflow.validateStringExpr` must handle `workflow.InputRef`.
- `internal/workflow.validateValueExpr` must handle `workflow.InputRef`.
- `internal/runtime.resolveStringExpr` must handle `workflow.InputRef`.
- `internal/runtime.resolveValueExpr` must handle `workflow.InputRef`.

`format(...)` arguments may contain `input(...)` directly or through a local
variable:

```python
feature_workspace = input("feature_dir")
spec_path = format("{dir}/spec.md", dir = feature_workspace)
```

## Starlark Loader Changes

### New Builtins

Add predeclared builtins:

- `input`
- `subworkflow`

`input(name)` returns an `inputValue` that wraps `workflow.InputRef`.
It should not validate that `name` appears in `workflow(inputs = [...])` during
Starlark evaluation because the `workflow(...)` call may not have happened yet.
That check belongs in the workflow validator after the full workflow tree has
been loaded.

`subworkflow(...)` returns a `subworkflowValue` that wraps
`workflow.Subworkflow`.
It should populate `Subworkflow.ModuleDir` from `currentCallerModulePath(thread)`.

### `workflow(...)`

Extend `workflow(...)` with optional fields:

- `inputs`
- `output_artifacts`
- `output_results`

Existing workflows without those fields remain valid.

### Loading Child Workflows

The loader should resolve child workflow paths with the same boundary rules used
for `load(...)`.

Recommended implementation:

1. `subworkflow(...)` records the raw workflow path and caller module directory.
2. After the parent Starlark file is evaluated, `Loader.Load` walks the workflow
   tree and resolves every subworkflow path.
3. The loader loads each child workflow file through a workflow-file loading
   path that allows top-level `wf`.
4. The loader attaches the loaded child workflow to `Subworkflow.Workflow`.
5. The workflow validator validates the fully loaded tree.

This keeps Starlark execution simple and keeps runtime execution independent
from Starlark file loading.

Child workflow loading should use a workflow-file stack:

```go
type workflowLoadContext struct {
    loading []string
}
```

The stack lives in a per-call load context, not on `Loader`, so concurrent
loads using the same `Loader` value do not share mutable state. Public
`Loader.Load(path)` should create a fresh `workflowLoadContext` and delegate to
an internal recursive method such as:

```go
func (l Loader) loadWorkflow(path string, ctx *workflowLoadContext, allowParam bool) (*workflow.Workflow, error)
```

`ctx.loading` contains canonical absolute workflow file paths currently being
loaded through `Loader.Load` and nested `subworkflow(...)` references. The same
`ctx` is passed into recursive child workflow loads. If a child path already
exists in `ctx.loading`, loading fails with a subworkflow cycle error.

Do not reuse one mutable `*workflow.Workflow` pointer for multiple
`subworkflow(...)` nodes. If the same child workflow file is referenced twice,
execute the child workflow file twice and attach a fresh workflow object to each
node. A source-file or parsed-file cache is acceptable later, but it must not
share mutable workflow model instances across subworkflow nodes.

Each workflow file evaluation should have its own Starlark helper-module cache
for `load(...)`. This avoids accidental pointer aliasing between separate
subworkflow instances. `load(...)` cycle detection remains separate from
subworkflow file cycle detection.

When loading a child workflow for `subworkflow(...)`, the load session should run
with `param(...)` disabled for the child workflow file and the helper modules it
loads. A good error shape is:

```text
load subworkflow "spec": param("name") is not allowed in subworkflows; declare input("name")
```

### Entry File Rule

Top-level entry workflow files still define `wf`.

Unlike ordinary loaded helper modules, a file referenced by `subworkflow(...)`
must define `wf`. It is a workflow file, not a helper module.

This means `subworkflow(...)` should not use Starlark `load(...)` internally.
It needs a separate child workflow loading path that allows top-level `wf`.

## Validation Changes

Validation should become scope-aware.

Current validation uses one global map of seen task IDs while traversing all
nodes. Subworkflows need separate internal scopes.

The `workflow.Validator` should not load Starlark files. It should validate the
already-loaded workflow tree. The `starlarkdsl.Loader` is responsible for
resolving subworkflow paths, loading child workflow files, attaching child
workflow objects to `Subworkflow` nodes, and detecting subworkflow file cycles.

Validator API should expose top-level input bindings without changing the
`Validate(wf *Workflow)` method shape:

```go
type Validator struct {
    BaseDir string
    Inputs  map[string]string
}
```

`Inputs` contains CLI input bindings for the top-level workflow. Existing tests
that construct workflows without `input(...)` values can keep using
`Validator{BaseDir: ...}`.

The validator uses `map[string]string` because top-level CLI input bindings are
strings and validation only needs key presence. Runtime uses `map[string]any`
because subworkflow input expressions may resolve from `json_ref(...)` and other
runtime values that are not necessarily strings.

Internally, validation should use an explicit scope object rather than one
global ID map:

```go
type nodeInfo struct {
    artifacts  map[string]struct{}
    resultKeys map[string]struct{}
}

type validationScope struct {
    seenNodes       map[string]nodeInfo
    allIDs          map[string]struct{}
    activeLoops     map[string]struct{}
    declaredInputs  map[string]struct{}
    availableInputs map[string]struct{}
    templateBaseDir  string
}
```

`seenNodes` is the unified lookup table for nodes that expose references:

- tasks register their artifact keys and result keys
- subworkflows register their child workflow output artifact keys and output
  result keys
- loops do not register in `seenNodes` because loops have no public artifact or
  result outputs in this design

`path_ref(...)` and `json_ref(...)` validation should both read from
`seenNodes`. There should not be one lookup path for tasks and a separate lookup
path for subworkflows.

`allIDs` is the workflow-scope uniqueness set for task IDs, loop IDs, and
subworkflow IDs. Each workflow gets its own `allIDs` and `seenNodes` maps. Loops
reuse the current workflow scope because step IDs remain unique within a
workflow, including nested loop bodies. Subworkflows create a new child workflow
scope, so parent and child task IDs may overlap.

Do not pass the parent workflow's global ID set into child workflow validation.
The current single-map approach is valid only before subworkflows exist.

Input set semantics:

- `declaredInputs` is the current workflow's `inputs = [...]` declaration.
- `availableInputs` is the set of values known to be provided for this workflow
  invocation.
- For a top-level workflow, `availableInputs` comes from `Validator.Inputs`.
- For a child workflow, `availableInputs` is initialized from the child's
  `declaredInputs`; the parent binding map is validated separately before the
  child can run.
- Validation fails if `declaredInputs` is not a subset of `availableInputs`.
- `input("x")` validation checks `declaredInputs`, so a workflow cannot use an
  undeclared input even if a value named `x` happens to be available.

For child workflow scopes, `availableInputs == declaredInputs` by construction,
so the subset check is intentionally a no-op. Parent binding completeness for a
child is enforced in the `Subworkflow` node validation step that checks the
parent `inputs` map against the child's declared inputs. Keeping
`availableInputs` in the child scope is only for uniform scope handling.

`templateBaseDir` is the fallback base directory for prompt templates whose
`Prompt.TemplateDir` is empty. For a child workflow, use the child workflow
file's directory, not the parent workflow file's directory. Prompt values
created through `template_file(...)` should still carry their exact module
directory in `Prompt.TemplateDir`, which remains the primary path resolution
mechanism.

### Parent Scope

In a parent workflow:

- task IDs, loop IDs, and subworkflow IDs are unique within that workflow scope
- a task may reference earlier parent tasks
- a task may reference earlier parent subworkflow public outputs
- a task may reference declared parent `input(...)` values
- a task may not reference child internal task IDs

### Child Scope

In a child workflow:

- internal task IDs and loop IDs are unique inside the child scope
- internal tasks may reference earlier internal tasks
- internal tasks may reference declared child `input(...)` values
- internal tasks may not reference parent task IDs directly
- child output expressions may reference internal tasks and child inputs

### Input Availability

The validator should receive the set of runtime inputs available to the workflow
scope it is validating.

For the top-level workflow, that set comes from CLI input binding keys.

For a child workflow, that set comes from the child's own `inputs` declaration.
The parent binding map is validated separately against the parent scope.

Validation should fail when:

- a top-level workflow declares an input that is missing from CLI input bindings
- a workflow uses `input("x")` without declaring `"x"` in `inputs`
- a parent subworkflow binding omits a child-declared input
- a parent subworkflow binding supplies a key the child did not declare

### Subworkflow Validation Steps

When validating a `Subworkflow` node:

1. Validate the node ID and path.
2. Require `Subworkflow.Workflow` to be non-nil.
3. Validate the child workflow in a new scope.
4. Validate the parent `inputs` map expressions against the parent scope,
   including the parent's active loop set.
5. Check that every child `inputs` declaration has exactly one parent binding.
6. Reject unknown input keys in the parent binding map.
7. Register the subworkflow node in the parent scope with:
   - artifact keys from child `output_artifacts`
   - result keys from child `output_results`

Child workflow validation must start with an empty active loop set. Parent loop
state is visible only while validating parent-side subworkflow input
expressions, not while validating the child internals.

### Forward References

Forward reference behavior stays the same:

- parent steps can reference only earlier parent-scope nodes
- child steps can reference only earlier child-scope nodes
- child output expressions can reference any step that is guaranteed to have
  executed by the end of the child workflow

Workflow output expressions should be validated against the final scope returned
after validating the workflow's `steps`. Because `repeat_until(max_iters >= 1)`
always executes its body at least once, the final scope includes tasks in loop
bodies using the same "latest successful execution" rule as normal downstream
references. If a future conditional step can skip its body entirely, output
validation must be revisited for that new control-flow primitive.

### Duplicate IDs

Parent and child scopes should not collide.

This is valid:

```python
wf = workflow(
    id = "parent",
    steps = [
        task(id = "write_spec", ...),
        subworkflow(id = "spec_refinement", workflow = "spec-refinement.star", inputs = {...}),
    ],
)
```

even if `spec-refinement.star` also contains an internal task with:

```python
task(id = "write_spec", ...)
```

Inside one workflow scope, duplicate IDs remain invalid.

### Cycles

The loader must reject subworkflow cycles before validation:

```text
subworkflow cycle detected:
  parent.star
  spec-refinement.star
  parent.star
```

The cycle cache should be separate from the Starlark `load(...)` module cycle
cache because these are workflow files, not helper modules.

## Runtime Changes

Add runtime inputs:

```go
type RunInput struct {
    Workflow     *workflow.Workflow
    WorkflowPath string
    BaseDir      string
    Workdir      string
    Inputs       map[string]any
}
```

For top-level runs, CLI input bindings populate `Inputs`.

For subworkflow runs, the parent runtime evaluates the subworkflow input
expressions and passes the resulting values into the child run.

### State

Add input values to runtime state:

```go
type state struct {
    artifacts map[string]map[string]string
    results   map[string]map[string]any
    loops     map[string]int
    inputs    map[string]any
}
```

`resolveValueExpr` and `resolveStringExpr` should resolve `workflow.InputRef`
from `st.inputs`.

### Subworkflow Execution

When the runtime encounters a `Subworkflow` node:

1. Require the node to have an attached child workflow.
2. Evaluate the node input expressions against the parent state.
3. Create a fresh child runtime state with the evaluated inputs.
4. Run the attached child workflow sequentially.
5. Evaluate the child `output_artifacts` and `output_results` against the child
   final state.
6. Register those evaluated outputs under the parent subworkflow ID:
   - `st.artifacts[subworkflow.ID] = childArtifactOutputs`
   - `st.results[subworkflow.ID] = childResultOutputs`
7. Continue parent execution.

The child should share:

- context
- executor map
- logger
- workdir
- base dir

The child should not share:

- task artifact state
- task result state
- active loop state
- input map

### Failure Behavior

If a child task fails, the parent run fails.

Error messages should include both the subworkflow ID and the child failing
step ID. Example:

```text
step spec.review_spec: artifact "review_report": expected file "..."
```

Use the existing flat `stepError` shape at the parent boundary:

```go
func qualifySubworkflowError(subworkflowID string, err error) error {
    childStepID, ok := errStepID(err)
    if !ok {
        return stepError{StepID: subworkflowID, Err: err}
    }
    return stepError{
        StepID: subworkflowID + "." + childStepID,
        Err:    unwrapStepError(err),
    }
}
```

The exact helper names can differ, but the parent should rewrite child
`stepError{StepID: "review_spec"}` into
`stepError{StepID: "spec.review_spec"}` before logging or returning it. That
keeps logger behavior compatible with the current flat step ID field.

For nested subworkflows, every runtime level applies the same qualification when
its direct child returns an error. For example, if child subworkflow `refine`
inside parent subworkflow `spec` fails at `inner_task`, the `refine` runtime
returns `refine.inner_task` to `spec`, and the `spec` runtime returns
`spec.refine.inner_task` to its parent.

### Logging

Add subworkflow progress events:

```text
[12:00:01] subworkflow start id=spec workflow=spec-refinement.star
[12:00:42] subworkflow done id=spec artifacts=spec outcome=stop
[12:00:42] subworkflow failed id=spec step=review_spec error=...
```

Child task logs may either:

- keep child task IDs unqualified and rely on surrounding subworkflow start/done
  events
- qualify child task IDs as `spec.review_spec`

Prefer qualified child task IDs if it is simple to implement.

## CLI Behavior

The clean CLI should use workflow inputs:

```sh
daiag run --workflow workflows/feature-development.star --input name=indicators
```

For top-level workflows:

- declared `inputs` are satisfied from `--input`
- `input("name")` resolves to the value from `--input name=...`
- missing declared inputs are validation errors before execution

Compatibility:

- keep `--param key=value` as an alias for `--input key=value` for existing
  scripts
- keep `param(...)` working for existing top-level workflows
- do not allow both `--input name=x` and `--param name=y` with different values
  in the same run

## Path Resolution

Subworkflow file paths should follow module-like path rules:

- relative to the Starlark file where `subworkflow(...)` appears
- canonicalized under the workflow base directory
- must end in `.star`
- no URLs
- no path escape through `..`

Artifact paths remain runtime workflow data:

- not resolved relative to the subworkflow file
- interpreted relative to workflow execution context and `--workdir`
- absolute artifact paths preserved as-is

Prompt template paths inside child workflows keep the existing rule:

- `template_file(...)` paths resolve relative to the module where
  `template_file(...)` appears

## Test Plan

Add loader and validator tests for:

- workflow with declared inputs loads successfully
- missing top-level input fails validation
- `param(...)` in a subworkflow fails
- duplicate workflow input declarations fail validation
- `input("x")` without `inputs = ["x"]` fails validation
- child workflow path outside base dir fails validation
- child workflow without top-level `wf` fails validation
- child workflow with no declared inputs can be called with omitted `inputs`
- child workflow with no declared inputs can be called with `inputs = {}`
- child workflow missing parent input binding fails validation
- parent binding with unknown child input key fails validation
- parent cannot reference child internal task IDs
- parent can reference child declared output artifacts and results
- duplicate IDs between parent and child are allowed
- duplicate IDs inside child still fail
- two subworkflow nodes referencing the same child file receive distinct
  workflow model instances
- subworkflow cycles fail with a clear error

Add runtime tests for:

- parent passes a literal input to a child workflow
- `format(...)` can use `input(...)` values as arguments
- parent passes `path_ref(...)` into a child workflow
- child output artifact is visible as `path_ref(subworkflow_id, key)`
- child output result is visible as `json_ref(subworkflow_id, key)`
- child output expressions may pass through declared `input(...)` values
- child runtime state does not leak internal task IDs to parent
- child failure is reported with subworkflow context
- nested child failure is reported with recursive subworkflow context such as
  `spec.refine.inner_task`

Add CLI-level tests for:

- `--input` satisfies top-level workflow `inputs`
- `--param` remains a compatibility alias for top-level workflow `inputs`
- conflicting `--input` and `--param` values for the same key fail

## Rollout Plan

1. Add model types for `InputRef`, `Subworkflow`, and workflow outputs.
2. Add Starlark values and builtins for `input(...)` and `subworkflow(...)`.
3. Extend `workflow(...)` unpacking for `inputs`, `output_artifacts`, and
   `output_results`.
4. Refactor validation into explicit workflow scopes and add `InputRef`
   validation.
5. Add runtime `InputRef` resolution and CLI `--input key=value` together; keep
   `--param key=value` as a compatibility alias.
6. Add child workflow loading and subworkflow cycle detection. This step also
   adds the `Subworkflow` case in workflow validation, because child workflows
   must be loaded before the validator can validate the child scope, check
   parent input bindings, and register the subworkflow's public output keys in
   the parent scope.
7. Add subworkflow execution and output registration.
8. Add logging events.
9. Update `docs/workflow-language.md` after implementation lands.
10. Convert one existing example, likely the development workflow, to use a
    `spec_refinement` subworkflow as the first real example.

## Open Questions

### Should `inputs` Allow Defaults?

Not in v1.

Defaults make validation and CLI behavior more complex. Require all declared
inputs to be provided.

### Should Inputs Be Typed?

Not in v1.

Most existing workflow data is path-like strings. Start with untyped runtime
values and add schema validation only when runtime behavior needs it.

### Should Child Workflows Inherit Parent `default_executor`?

No.

A child workflow should be self-contained. It either declares its own
`default_executor` or each task declares an executor.

### Should Parent Workflows Override Child Executors?

Not in v1.

Executor override is useful, but it creates another policy layer. Keep the first
implementation explicit.

### Should We Add Task-Level Inputs?

No for v1.

Reusable task constructors should accept dynamic values as Starlark function
arguments. Adding `inputs = ...` to `task(...)` creates another contract beside
`template_file(vars = {...})`, `artifacts = {...}`, and `result_keys`.

Subworkflow inputs are different: they define a component boundary and are
needed for validation and composition.

## Related Documents

- `docs/subworkflow-implementation-tasks.md`
