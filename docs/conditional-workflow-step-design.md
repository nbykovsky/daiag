# Design: Conditional Workflow Step

## Goal

Add runtime conditional execution to the workflow DSL so a workflow can choose
between step groups from a predicate over earlier workflow data.

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
    else_steps = [
        task(
            id = "record_no_repair_needed",
            prompt = "Write docs/features/indicators/repair_status.md saying no code repair was needed. Return JSON with outcome.",
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
- `else` is also a Starlark keyword, so the optional false branch should be
  named `else_steps`, not `else`.
- Starlark `if` executes at workflow load time, before runtime task results
  exist.
- The feature wraps one or more workflow steps; it is not an executor task by
  itself.

## V1 Scope

The first implementation should be intentionally narrow:

- add `when(id, condition, steps, else_steps = [])`
- support the existing predicate type, currently `eq(...)`
- support `task(...)`, `repeat_until(...)`, `subworkflow(...)`, and nested
  `when(...)` values inside `steps` and `else_steps`
- evaluate `condition` once before choosing a branch
- run `steps` sequentially when the predicate is true
- run `else_steps` sequentially when the predicate is false and `else_steps` is
  provided
- skip the conditional and continue the parent workflow when the predicate is
  false and `else_steps` is omitted or empty
- allow `when(...)` inside `repeat_until(...)` loops

Do not add these in v1:

- branch output declarations
- path or JSON references to branch-internal steps from later parent steps
- dynamic task fan-out
- richer predicates such as `and`, `or`, `not`, or `ne`

The no-output rule is the main simplifier. It prevents later steps from reading
artifacts or results that may not exist because only one branch ran.

Empty `steps` and `else_steps` lists are allowed. A true condition with
`steps = []`, or a false condition with no `else_steps`, is a no-op and
continues the parent workflow. This matches the current implicit allowance for
empty `repeat_until(...)` bodies.

## Runtime Semantics

For each `when(...)` node:

1. Resolve and evaluate `condition` against the current runtime state.
2. If evaluation fails, fail the workflow with the `when(...)` node ID.
3. If the predicate is true, run `steps` in order using the same runtime state
   as the parent workflow.
4. If the predicate is false, run `else_steps` in order using the same runtime
   state as the parent workflow.
5. If the predicate is false and `else_steps` is empty, continue without
   running branch steps.
6. If a nested branch step fails, return that nested step error directly.

Nested steps use the same artifact, result, loop, input, workdir, executor, and
subworkflow behavior they use elsewhere.

Inside a `repeat_until(...)` loop, `when(...)` is evaluated on every loop
iteration. A different branch may run on a later iteration. V1 avoids stale data
by making branch-internal step IDs invisible outside the branch.

## Reference Visibility

Validation should distinguish ID uniqueness from reference visibility.

Step IDs remain unique across the whole workflow scope, including nested loops
and conditional branches. This matches the existing duplicate-ID behavior and
keeps logs unambiguous.

Reference visibility is narrower:

- `condition` may reference only steps visible before the `when(...)` node.
- steps inside `steps` may reference steps visible before the `when(...)` node
  and earlier steps inside `steps`.
- steps inside `else_steps` may reference steps visible before the `when(...)`
  node and earlier steps inside `else_steps`.
- `steps` and `else_steps` may not cross-reference each other because only one
  branch runs.
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
- unsupported node type in `steps` or `else_steps`
- any existing task, loop, subworkflow, expression, or template validation error
  inside either branch
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
    ElseSteps []Node
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
- update all `unpackSteps` call sites to pass the field name `steps`
- update error text from `task, repeat_until, or subworkflow` to include
  `when`
- update `loadSubworkflowsInNode` to recurse through `When.Steps` and
  `When.ElseSteps`
- update step unpacking to accept a field name so `else_steps = "bad"` reports
  `else_steps must be a list`
- update predicate unpacking so `when(condition = "bad", ...)` reports
  `condition must be a predicate` instead of `until must be a predicate`

The builtin should unpack:

```go
"id", &id,
"condition", &conditionValue,
"steps", &stepsValue,
"else_steps?", &elseStepsValue,
```

Change `unpackPredicate` to accept a field name:

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

Change `unpackSteps` to accept a field name, then add an optional steps helper
for `else_steps`:

```go
func unpackSteps(value starlark.Value, field string) ([]workflow.Node, error) {
    list, ok := value.(*starlark.List)
    if !ok {
        return nil, fmt.Errorf("%s must be a list", field)
    }

    steps := make([]workflow.Node, 0, list.Len())
    for i := 0; i < list.Len(); i++ {
        item := list.Index(i)
        switch value := item.(type) {
        case *taskValue:
            steps = append(steps, value.task)
        case *repeatUntilValue:
            steps = append(steps, value.loop)
        case *subworkflowValue:
            steps = append(steps, value.subworkflow)
        case *whenValue:
            steps = append(steps, value.when)
        default:
            return nil, fmt.Errorf("%s[%d] must be a task, repeat_until, subworkflow, or when, got %s", field, i, item.Type())
        }
    }
    return steps, nil
}
```

```go
func unpackOptionalSteps(value starlark.Value, field string) ([]workflow.Node, error) {
    if value == starlark.None {
        return nil, nil
    }
    return unpackSteps(value, field)
}
```

Use `unpackSteps(stepsValue, "steps")` for the required true branch and
`unpackOptionalSteps(elseStepsValue, "else_steps")` for the false branch.

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
    if _, err := v.validateSteps(n.ElseSteps, current, allIDs, defaultExecutor, activeLoops, declaredInputs, templateBaseDir); err != nil {
        return nil, err
    }
    // Intentionally no current update here: conditional branches do not publish
    // branch artifacts or results to later parent steps.
    continue
```

Validate the condition before the branch steps and against `current`, the
pre-branch scope. This intentionally differs from `repeat_until(...)`, which
validates `until` after body validation and against the post-body scope because
`until` runs after the loop body.

Do not assign either returned branch scope back to `current`, and do not add a
`current[n.ID] = ...` entry for the `when(...)` node.

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
    if ok {
        if e.Logger != nil {
            e.Logger.WhenCheck(node.ID, "steps")
        }
        return e.runNodes(ctx, input, st, node.Steps)
    }
    if len(node.ElseSteps) == 0 {
        if e.Logger != nil {
            e.Logger.WhenCheck(node.ID, "skip")
        }
        return nil
    }
    if e.Logger != nil {
        e.Logger.WhenCheck(node.ID, "else_steps")
    }
    return e.runNodes(ctx, input, st, node.ElseSteps)
}
```

Nested step failures should keep their own step IDs, matching
`repeat_until(...)` body behavior.

Branch steps that run will still write their artifacts and results into the
shared runtime state. That is acceptable in v1 because validation prevents
later parent steps from referencing branch-internal IDs. The implementation does
not need runtime cleanup when the other branch runs on a later loop iteration.

### 5. Logging

Update `internal/logging/logger.go` with one concise line:

```go
func (l *Logger) WhenCheck(id, result string) {
    l.printf("when check id=%s result=%s", id, result)
}
```

This mirrors `loop check id=<id> result=<stop|continue>`. Valid `result`
values are `steps`, `else_steps`, and `skip`.

### 6. Tests

Add focused tests:

- `internal/starlarkdsl`: loads a workflow with `when(...)`
- `internal/starlarkdsl`: rejects a parent step that references a branch-internal
  step after `when(...)`
- `internal/starlarkdsl`: rejects a `when(...)` condition that references a
  branch-internal step
- `internal/starlarkdsl`: rejects cross-references between `steps` and
  `else_steps`
- `internal/starlarkdsl`: loads a subworkflow nested inside `when(...)`
- `internal/starlarkdsl`: rejects duplicate step IDs across parent, `steps`,
  and `else_steps`
- `internal/runtime`: runs branch steps when the condition is true
- `internal/runtime`: runs `else_steps` when the condition is false
- `internal/runtime`: skips the conditional when the condition is false and
  `else_steps` is empty
- `internal/runtime`: evaluates `when(...)` on each `repeat_until(...)`
  iteration
- `internal/runtime`: attributes condition evaluation failure to the
  `when(...)` ID
- `internal/logging`: logs `when check id=<id> result=steps`,
  `result=else_steps`, and `result=skip`

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

A later design can add branch outputs if concrete workflows need values from a
conditional after it completes. That should be a separate merge contract, either
by extending `when(...)` with explicit output declarations or by adding a
separate `branch(...)` node. It should define how outputs are resolved for both
the true and false branches, and should not be bundled into the first
conditional execution change.
