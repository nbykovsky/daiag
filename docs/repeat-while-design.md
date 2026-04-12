# Design: `repeat_while(...)`

## Goal

Add a pre-condition loop to the workflow DSL so a workflow can evaluate current
state before running repair or address steps.

This covers cases where a workflow should skip work entirely when the current
state is already acceptable:

- run a code repair step only while QA triage reports code issues
- run a review addresser only while review output reports addressable concerns
- re-check state before every repair pass, including before the first pass
- avoid no-op tasks whose only purpose is to simulate a pre-condition loop

## Proposed DSL Surface

Use a `repeat_while(...)` builtin. It complements `repeat_until(...)` rather
than replacing it.

```python
repeat_while(
    id = "repair_while_needed",
    max_iters = 3,
    check_steps = [
        task(
            id = "qa_triage",
            prompt = template_file(
                "../agents/qa-triage.md",
                vars = {
                    "QA_STATUS_PATH": format(
                        "docs/features/indicators/qa-triage-{iter}.md",
                        iter = loop_iter("repair_while_needed"),
                    ),
                },
            ),
            artifacts = {
                "status": artifact(format(
                    "docs/features/indicators/qa-triage-{iter}.md",
                    iter = loop_iter("repair_while_needed"),
                )),
            },
            result_keys = ["outcome"],
        ),
    ],
    condition = eq(json_ref("qa_triage", "outcome"), "code_issues"),
    steps = [
        task(
            id = "repair_code",
            prompt = template_file(
                "../agents/code-repair.md",
                vars = {
                    "QA_STATUS_PATH": path_ref("qa_triage", "status"),
                    "REPAIR_STATUS_PATH": format(
                        "docs/features/indicators/repair-{iter}.md",
                        iter = loop_iter("repair_while_needed"),
                    ),
                },
            ),
            artifacts = {
                "status": artifact(format(
                    "docs/features/indicators/repair-{iter}.md",
                    iter = loop_iter("repair_while_needed"),
                )),
            },
            result_keys = ["outcome"],
        ),
    ],
)
```

Required fields:

- `id`
- `max_iters`
- `check_steps`
- `condition`
- `steps`

`check_steps` is explicit because runtime predicates do not execute work. They
only inspect existing workflow state. A useful pre-condition loop usually needs
a check task to produce the state that the condition reads before each body
pass.

## Relationship To Existing Constructs

`repeat_until(...)` is a do-until loop:

```text
run body
check whether to stop
```

`repeat_while(...)` is a pre-check loop:

```text
run check
if condition is false, stop
run body
repeat
```

`when(...)` can guard a one-time branch, and `when(...)` around
`repeat_until(...)` can approximate some pre-check flows. `repeat_while(...)`
is useful when the workflow needs to re-check before every body execution and
skip the body when the initial check is already clean.

## V1 Scope

The first implementation should be intentionally narrow:

- add `repeat_while(id, max_iters, check_steps, condition, steps)`
- support the existing predicate type, currently `eq(...)`
- support `task(...)`, `repeat_until(...)`, `repeat_while(...)`,
  `when(...)`, and `subworkflow(...)` values inside `check_steps` and `steps`
- require `check_steps` to contain at least one step
- evaluate `condition` after `check_steps` and before `steps`
- allow zero body executions when the first condition evaluation is false
- run at most `max_iters` body executions
- run one final check after the `max_iters` body execution to determine whether
  the loop can stop cleanly
- expose final `check_steps` artifacts and results to later parent steps
- keep body step artifacts and results invisible to later parent steps because
  the body may run zero times

Do not add these in v1:

- explicit loop output declarations
- path or JSON references to body-internal steps from later parent steps
- `break` or `continue`
- dynamic task fan-out
- richer predicates such as `and`, `or`, `not`, or `ne`
- sleep, polling intervals, or asynchronous waiting

The output rule is the main simplifier. Check steps are guaranteed to run at
least once, so their final artifacts and results can be safely referenced after
the loop. Body steps are not guaranteed to run, so later parent steps cannot
reference them.

## Runtime Semantics

For each `repeat_while(...)` node:

1. Set the loop iteration value for `loop_iter(id)`.
2. Run `check_steps` in order.
3. Evaluate `condition` against the current runtime state.
4. If condition evaluation fails, fail the workflow with the `repeat_while(...)`
   node ID.
5. If the predicate is false, stop the loop successfully.
6. If the predicate is true and the body has already run `max_iters` times,
   fail the workflow with the `repeat_while(...)` node ID.
7. If the predicate is true and the body has run fewer than `max_iters` times,
   run `steps` in order.
8. Repeat from step 1.

`max_iters` counts body executions, not check executions.

For example, with `max_iters = 3`:

- if the first check is false, the body runs `0` times and the loop succeeds
- if checks 1, 2, and 3 are true, the body runs `3` times
- after the third body run, check 4 runs
- if check 4 is false, the loop succeeds
- if check 4 is true, the loop fails because the condition still requires work
  after the maximum body executions

`loop_iter(loop_id)` is active during both `check_steps` and `steps`.
The body uses the same iteration number as the check that allowed it to run.
A final check after the last allowed body run uses `max_iters + 1`.

Nested step failures keep their own step IDs, matching `repeat_until(...)` body
behavior.

## Reference Visibility

Validation should distinguish ID uniqueness from reference visibility.

Step IDs remain unique across the whole workflow scope, including nested loops,
conditionals, `check_steps`, and body steps. This keeps logs and runtime result
maps unambiguous.

Reference visibility should be:

- `check_steps` may reference steps visible before the `repeat_while(...)` node
  and earlier steps inside `check_steps`
- `condition` may reference steps visible before the `repeat_while(...)` node
  and steps from `check_steps`
- body `steps` may reference steps visible before the `repeat_while(...)` node,
  steps from `check_steps`, and earlier body steps in the same body
- `check_steps` and `condition` may not reference body steps because the body
  has not run before the first condition evaluation
- later parent steps and workflow outputs may reference `check_steps` artifacts
  and results
- later parent steps and workflow outputs may not reference body step artifacts
  or results
- the `repeat_while(...)` node itself has no artifacts or results, so
  `path_ref("repair_while_needed", "...")` and
  `json_ref("repair_while_needed", "...")` are invalid

If a later parent step needs a durable body artifact in v1, the workflow should
write to a stable path known outside the loop, or the feature should wait for a
future explicit loop output contract.

## Validation Rules

Add validation errors for:

- duplicate `repeat_while(...)` ID or duplicate nested step ID
- `max_iters < 1`
- missing or empty `check_steps`
- missing or unsupported `condition`
- unsupported node type in `check_steps` or `steps`
- `condition` referencing an unknown step or a body-internal step
- `check_steps` referencing a body-internal step
- later parent steps referencing body-internal artifacts or results
- workflow outputs referencing body-internal artifacts or results
- any existing task, loop, subworkflow, expression, or template validation error
  inside `check_steps` or `steps`

Empty `repeat_while(...)` IDs require no special validation code. The existing
generic node ID check in `validateSteps` should catch them as `step ID is
empty`.

`loop_iter(...)` should be valid in `check_steps`, `condition`, and body
`steps` only when it references the structurally enclosing
`repeat_while(...)` loop or another active outer loop.

## Implementation Plan

### 1. Workflow model

Update `internal/workflow/model.go`:

```go
type RepeatWhile struct {
    ID         string
    MaxIters   int
    CheckSteps []Node
    Condition  Predicate
    Steps      []Node
}

func (*RepeatWhile) node() {}

func (r *RepeatWhile) NodeID() string {
    return r.ID
}
```

### 2. Starlark DSL

Update `internal/starlarkdsl`:

- add `repeat_while` to `Loader.predeclared`
- add `builtinRepeatWhile`
- add `repeatWhileValue`
- update `unpackSteps` to accept `*repeatWhileValue`
- update `loadSubworkflowsInNode` to recurse through
  `RepeatWhile.CheckSteps` and `RepeatWhile.Steps`
- ensure field-specific errors report `check_steps must be a list`,
  `steps must be a list`, and `condition must be a predicate`

The builtin should unpack:

```go
"id", &id,
"max_iters", &maxIters,
"check_steps", &checkStepsValue,
"condition", &conditionValue,
"steps", &stepsValue,
```

It should call:

```go
checkSteps, err := unpackSteps(checkStepsValue, "check_steps")
condition, err := unpackPredicate(conditionValue, "condition")
steps, err := unpackSteps(stepsValue, "steps")
```

### 3. Workflow validation

Update `internal/workflow/validate.go` in `validateSteps`:

```go
case *RepeatWhile:
    if n.MaxIters < 1 {
        return nil, fmt.Errorf("repeat_while %q: max_iters must be at least 1", n.ID)
    }
    if len(n.CheckSteps) == 0 {
        return nil, fmt.Errorf("repeat_while %q: check_steps are required", n.ID)
    }

    loopScope := cloneStringSet(activeLoops)
    loopScope[n.ID] = struct{}{}

    checkSeen, err := v.validateSteps(n.CheckSteps, current, allIDs, defaultExecutor, loopScope, declaredInputs, templateBaseDir)
    if err != nil {
        return nil, err
    }
    if err := v.validatePredicate(n.Condition, checkSeen, loopScope, declaredInputs); err != nil {
        return nil, fmt.Errorf("repeat_while %q: %w", n.ID, err)
    }
    if _, err := v.validateSteps(n.Steps, checkSeen, allIDs, defaultExecutor, loopScope, declaredInputs, templateBaseDir); err != nil {
        return nil, err
    }

    current = checkSeen
```

This intentionally differs from `repeat_until(...)`:

- `repeat_until(...)` validates `until` after body validation because the body
  runs before the predicate
- `repeat_while(...)` validates `condition` after `check_steps` and before
  body validation because the check runs before the body
- `repeat_while(...)` publishes only `check_steps` to the parent scope because
  body steps may not run

### 4. Runtime engine

Update `internal/runtime/engine.go`:

- add a `*workflow.RepeatWhile` case in `runNodes`
- add `runRepeatWhile`

Suggested control flow:

```go
func (e Engine) runRepeatWhile(ctx context.Context, input RunInput, st *state, loop *workflow.RepeatWhile) error {
    defer delete(st.loops, loop.ID)

    bodyRuns := 0
    for {
        iter := bodyRuns + 1
        st.loops[loop.ID] = iter

        if e.Logger != nil {
            e.Logger.LoopIter(loop.ID, iter)
        }

        if err := e.runNodes(ctx, input, st, loop.CheckSteps); err != nil {
            return err
        }

        ok, err := evalPredicate(loop.Condition, st)
        if err != nil {
            return stepError{StepID: loop.ID, Err: err}
        }
        if !ok {
            if e.Logger != nil {
                e.Logger.LoopCheck(loop.ID, "stop")
            }
            return nil
        }
        if bodyRuns >= loop.MaxIters {
            if e.Logger != nil {
                e.Logger.LoopCheck(loop.ID, "max_iters")
            }
            return stepError{
                StepID: loop.ID,
                Err: fmt.Errorf(
                    "loop reached max_iters=%d while condition remained true",
                    loop.MaxIters,
                ),
            }
        }

        if e.Logger != nil {
            e.Logger.LoopCheck(loop.ID, "run")
        }
        if err := e.runNodes(ctx, input, st, loop.Steps); err != nil {
            return err
        }
        bodyRuns++
    }
}
```

Valid `LoopCheck` result values for `repeat_while(...)` would be `run`, `stop`,
and `max_iters`. This reuses the existing concise loop logging format without
adding a second logger method.

### 5. Tests

Add focused tests:

- `internal/starlarkdsl`: loads a workflow with `repeat_while(...)`
- `internal/starlarkdsl`: rejects `repeat_while(max_iters < 1)`
- `internal/starlarkdsl`: rejects empty `check_steps`
- `internal/starlarkdsl`: rejects a condition that references a body step
- `internal/starlarkdsl`: rejects a later parent step that references a body
  step
- `internal/starlarkdsl`: allows a later parent step to reference a check step
- `internal/starlarkdsl`: allows `loop_iter(...)` in check and body artifacts
- `internal/runtime`: skips the body when the first condition is false
- `internal/runtime`: runs check, body, check, then stops when the second check
  is false
- `internal/runtime`: fails when the condition remains true after `max_iters`
  body executions and the final check
- `internal/runtime`: exposes final check step artifacts and results after the
  loop
- `internal/logging`: logs `loop check id=<id> result=run`,
  `result=stop`, and `result=max_iters`

Suggested package checks after implementation:

```sh
go test ./internal/workflow ./internal/starlarkdsl ./internal/runtime ./internal/logging
go test ./internal/cli
go build ./cmd/daiag
```

## Documentation Updates After Implementation

When the feature is implemented, update `docs/workflow-language.md` to describe
`repeat_while(...)` as supported syntax. Do not update the language reference
before the implementation lands because that file is the current implementation
reference.

The development workflow example can then replace no-op addresser and repair
patterns where a pre-check can skip the body safely.

## Future Work

A later design can add explicit loop outputs if concrete workflows need values
from body steps after the loop. That output contract should define what happens
when the body runs zero times, when it runs multiple times, and whether the
final value comes from the last body execution or from a fallback expression.
