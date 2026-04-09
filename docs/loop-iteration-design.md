# Loop Iteration Design

## Problem

In the current workflow model, tasks inside `repeat_until` typically reuse the same artifact paths on every iteration.

Example:

- iteration 1 writes `review.txt`
- iteration 2 writes `review.txt` again
- iteration 3 writes `review.txt` again

This is correct for mutable working files such as `poem.md`, but it is not ideal for diagnostic or review artifacts that should be preserved per iteration.

## Goal

Allow tasks inside `repeat_until` to access the current loop iteration number and use it in prompt variables and artifact paths.

This should make it possible to write files such as:

- `review-1.txt`
- `review-2.txt`
- `review-3.txt`

while still keeping the final artifact reference semantics simple.

## Recommendation

Add an explicit runtime expression:

```python
loop_iter("extend_until_ready")
```

This returns the current iteration number for the named loop.

The value should be:

- available only during execution inside that loop
- `1`-based, not `0`-based

`1`-based numbering matches current progress output such as `n=1` and produces more natural filenames.

## Example

Current review path:

```python
"REVIEW_PATH": format("{dir}/review.txt", dir = feature_dir)
```

Proposed review path:

```python
"REVIEW_PATH": format(
    "{dir}/review-{iter}.txt",
    dir = feature_dir,
    iter = loop_iter("extend_until_ready"),
)
```

Artifact declaration:

```python
artifacts = {
    "review": artifact(
        format(
            "{dir}/review-{iter}.txt",
            dir = feature_dir,
            iter = loop_iter("extend_until_ready"),
        )
    ),
}
```

## Why Explicit Is Better

The DSL already uses explicit symbolic references:

- `path_ref(step_id, artifact_key)`
- `json_ref(step_id, field)`

`loop_iter(loop_id)` fits that style.

An implicit variable such as `loop.i` would be harder to model cleanly because task definitions are built at workflow load time, while iteration values exist only at runtime.

## Required DSL Changes

Add one new builtin:

- `loop_iter(loop_id)`

It should return a symbolic runtime expression, not an integer computed at load time.

## Required Runtime Changes

The runtime must track the active iteration count for each executing `repeat_until` block.

At minimum:

- when a loop starts iteration 1, `loop_iter("<id>")` resolves to `1`
- on the next iteration it resolves to `2`
- nested loops should each have their own independent counters

## `format(...)` Implication

This feature also means `format(...)` can no longer be treated as purely load-time string interpolation.

Today `format(...)` is effectively static.
With `loop_iter(...)`, `format(...)` must support runtime expression values.

That suggests the model should treat `format(...)` as a string expression node rather than immediately resolving it during Starlark evaluation.

## Semantics After Loop Completion

Per-iteration files should remain on disk.

Example:

- `review-1.txt`
- `review-2.txt`
- `review-3.txt`

For downstream workflow references, the latest successful task execution should still win.

So after the loop completes:

- `path_ref("review_poem", "review")` should resolve to the final iteration's review file
- earlier iteration files remain available on disk for inspection

## What Should Stay Unchanged

This feature should not force every loop artifact to become iteration-specific.

Some artifacts are intentionally stable and should continue to reuse a single path:

- `poem.md`

The new capability should be optional and only used when a task needs per-iteration outputs.

## Validation Rules

Validation should reject:

- `loop_iter("")`
- references to unknown loop IDs
- use of `loop_iter("x")` outside the execution scope where loop `x` is active, if the runtime cannot resolve it

The validator should also ensure that a task only refers to loops that structurally enclose that task.

## Suggested Scope

Implement this as a narrow feature:

1. add `loop_iter(loop_id)`
2. make `format(...)` runtime-aware
3. keep all other DSL concepts unchanged

This is enough to solve per-iteration artifact naming without introducing a more general variable system.
