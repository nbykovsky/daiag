# Design: Conditional Workflow Step

## Goal

Add runtime conditional execution to the workflow DSL so a workflow can run a
step group only when a predicate over earlier workflow data is true.

This covers cases already called out by the development workflow example:

- run a review addresser only when review output reports addressable concerns
- run a code repair stage only when QA triage reports code issues
- keep `repeat_until(...)` loop bodies explicit without forcing no-op tasks

## Proposed DSL Surface

Use a `when(...)` builtin instead of `if`.

```python
when(
    id = "address_code_issues",
    condition = eq(json_ref("qa_triage", "outcome"), "code_issues"),
    steps = [
        task(
            id = "repair_code",
            prompt = template_file(
                "../agents/code-refiner.md",
                vars = {
                    "QA_STATUS_PATH": path_ref("qa_triage", "status"),
                    "REPAIR_STATUS_PATH": "docs/features/indicators/repair_status.md",
                },
            ),
            artifacts = {
                "status": artifact("docs/features/indicators/repair_status.md"),
            },
            result_keys = ["outcome"],
        ),
    ],
)
```

`if` should not be used as the builtin name:

- `if` is a Starlark keyword, so it is not available as a normal DSL builtin.
- Starlark `if` executes at workflow load time, before runtime task results
  exist.
- The feature wraps one or more workflow steps; it is not an executor task by
  itself.

## V1 Scope

The first implementation should be intentionally narrow:

- add `when(id, condition, steps)`
- support the existing predicate type, currently `eq(...)`
- support `task(...)`, `repeat_until(...)`, `subworkflow(...)`, and nested
  `when(...)` values inside `steps`
- evaluate `condition` once before running `steps`
- run the nested steps sequentially when the predicate is true
- skip the nested steps and continue the parent workflow when the predicate is
  false
- allow `when(...)` inside `repeat_until(...)` loops

Do not add these in v1:

- `else_steps`
- branch output declarations
- path or JSON references to skipped branch steps from later parent steps
- dynamic task fan-out
- richer predicates such as `and`, `or`, `not`, or `ne`

The no-output rule is the main simplifier. It prevents later steps from reading
artifacts or results that may not exist because the branch was skipped.

## Runtime Semantics

For each `when(...)` node:

1. Resolve and evaluate `condition` against the current runtime state.
2. If evaluation fails, fail the workflow with the `when(...)` node ID.
3. If the predicate is false, skip the nested steps and continue.
4. If the predicate is true, run the nested steps in order using the same
   runtime state as the parent workflow.
5. If a nested step fails, return that nested step error directly.

Nested steps use the same artifact, result, loop, input, workdir, executor, and
subworkflow behavior they use elsewhere.

Inside a `repeat_until(...)` loop, `when(...)` is evaluated on every loop
iteration. If it is skipped on a later iteration, branch outputs from an earlier
iteration must not become visible to later parent steps. V1 avoids stale data by
making branch-internal step IDs invisible outside the branch.

## Reference Visibility

Validation should distinguish ID uniqueness from reference visibility.

Step IDs remain unique across the whole workflow scope, including nested loops
and conditional branches. This matches the existing duplicate-ID behavior and
keeps logs unambiguous.

Reference visibility is narrower:

- `condition` may reference only steps visible before the `when(...)` node.
- steps inside the branch may reference steps visible before the `when(...)`
  node and earlier steps inside the same branch.
- steps after the `when(...)` node may not reference branch-internal step
  artifacts or results.
- workflow `output_artifacts` and `output_results` may not reference
  branch-internal step artifacts or results.
- the `when(...)` node itself has no artifacts or results, so
  `path_ref("address_code_issues", "...")` and
  `json_ref("address_code_issues", "...")` are invalid.

If a workflow needs a value after a conditional branch in v1, use a stable path
that is not declared as a branch artifact, or keep the existing always-run
no-op task pattern until branch outputs are designed.

## Validation Rules

Add validation errors for:

- duplicate `when(...)` ID or duplicate nested step ID
- missing or unsupported `condition`
- `condition` referencing an unknown or branch-internal step
- unsupported node type in `steps`
- any existing task, loop, subworkflow, expression, or template validation error
  inside the branch
- later parent steps referencing branch-internal artifacts or results
- workflow outputs referencing branch-internal artifacts or results

Empty `when(...)` IDs require no special validation code. The existing generic
node ID check in `validateSteps` should catch them as `step ID is empty`.

`loop_iter(...)` should be valid in a `when(...)` condition only when the
conditional node is structurally inside the referenced loop. This should require
no special case because `validatePredicate` already calls `validateValueExpr`,
and the existing `activeLoops` map flows from the enclosing validation context.

## Implementation Plan

### 1. Workflow model

Update `internal/workflow/model.go`:

```go
type When struct {
    ID        string
    Condition Predicate
    Steps     []Node
}

func (*When) node() {}

func (w *When) NodeID() string {
    return w.ID
}
```

### 2. Starlark DSL

Update `internal/starlarkdsl`:

- add `when` to `Loader.predeclared`
- add `builtinWhen`
- add `whenValue`
- update `unpackSteps` to accept `*whenValue`
- update error text from `task, repeat_until, or subworkflow` to include
  `when`
- update `loadSubworkflowsInNode` to recurse through `When.Steps`
- update predicate unpacking so `when(condition = "bad", ...)` reports
  `condition must be a predicate` instead of `until must be a predicate`

The builtin should unpack:

```go
"id", &id,
"condition", &conditionValue,
"steps", &stepsValue,
```

Prefer changing `unpackPredicate` to accept a field name:

```go
func unpackPredicate(value starlark.Value, field string) (workflow.Predicate, error) {
    predicate, ok := value.(*predicateValue)
    if !ok {
        return nil, fmt.Errorf("%s must be a predicate", field)
    }
    return predicate.predicate, nil
}
```

Then call `unpackPredicate(untilValue, "until")` from
`builtinRepeatUntil` and `unpackPredicate(conditionValue, "condition")` from
`builtinWhen`.

### 3. Workflow validation

Update `internal/workflow/validate.go` in `validateSteps`:

```go
case *When:
    if err := v.validatePredicate(n.Condition, current, activeLoops, declaredInputs); err != nil {
        return nil, fmt.Errorf("when %q: %w", n.ID, err)
    }
    if _, err := v.validateSteps(n.Steps, current, allIDs, defaultExecutor, activeLoops, declaredInputs, templateBaseDir); err != nil {
        return nil, err
    }
    // Do not merge the returned branch scope into current.
    // Branch outputs are intentionally invisible after the conditional.
```

Do not assign the returned branch scope back to `current`.

That preserves internal validation and duplicate-ID checks while preventing
later parent steps from referencing branch-internal outputs.

### 4. Runtime engine

Update `internal/runtime/engine.go`:

- add a `*workflow.When` case in `runNodes`
- add `runWhen`

```go
func (e Engine) runWhen(ctx context.Context, input RunInput, st *state, node *workflow.When) error {
    ok, err := evalPredicate(node.Condition, st)
    if err != nil {
        return stepError{StepID: node.ID, Err: err}
    }
    if e.Logger != nil {
        e.Logger.WhenCheck(node.ID, ok)
    }
    if !ok {
        return nil
    }
    return e.runNodes(ctx, input, st, node.Steps)
}
```

Nested step failures should keep their own step IDs, matching
`repeat_until(...)` body behavior.

Branch steps that run will still write their artifacts and results into the
shared runtime state. That is acceptable in v1 because validation prevents
later parent steps from referencing branch-internal IDs. The implementation does
not need runtime cleanup for skipped branches.

### 5. Logging

Update `internal/logging/logger.go` with one concise line:

```go
func (l *Logger) WhenCheck(id string, ok bool) {
    result := "skip"
    if ok {
        result = "run"
    }
    l.printf("when check id=%s result=%s", id, result)
}
```

This mirrors `loop check id=<id> result=<stop|continue>`.

### 6. Tests

Add focused tests:

- `internal/starlarkdsl`: loads a workflow with `when(...)`
- `internal/starlarkdsl`: rejects a parent step that references a branch-internal
  step after `when(...)`
- `internal/starlarkdsl`: rejects a `when(...)` condition that references a
  branch-internal step
- `internal/starlarkdsl`: loads a subworkflow nested inside `when(...)`
- `internal/starlarkdsl`: rejects duplicate step IDs across parent and branch
- `internal/runtime`: runs branch steps when the condition is true
- `internal/runtime`: skips branch steps when the condition is false
- `internal/runtime`: evaluates `when(...)` on each `repeat_until(...)`
  iteration
- `internal/runtime`: attributes condition evaluation failure to the
  `when(...)` ID
- `internal/logging`: logs `when check id=<id> result=run` and `result=skip`

Suggested package checks after implementation:

```sh
go test ./internal/workflow ./internal/starlarkdsl ./internal/runtime ./internal/logging
go test ./internal/cli
go build ./cmd/daiag
```

## Documentation Updates After Implementation

When the feature is implemented, update the language reference to describe
`when(...)` as supported syntax. Do not update `docs/workflow-language.md`
before the implementation lands because that file is the current implementation
reference.

The development workflow example can then replace no-op task workarounds where
the branch does not need to publish artifacts or results to later parent steps.

## Future Work

A later design can add a full branching node if concrete workflows need values
from both sides of a conditional:

```python
branch(
    id = "qa_repair",
    condition = eq(json_ref("qa_triage", "outcome"), "code_issues"),
    then_steps = [...],
    else_steps = [...],
    output_artifacts = {
        "status": path_ref("repair_code", "status"),
    },
    output_results = {
        "outcome": json_ref("repair_code", "outcome"),
    },
)
```

That feature needs a separate merge contract for branch outputs and should not
be bundled into the first conditional execution change.
