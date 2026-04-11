# Starlark `load()` Design

## Problem

Small workflows fit comfortably in one `.star` file.
Larger workflows do not.

As the DSL grows, a single workflow file starts to mix:

- path definitions
- executor defaults
- task constructors
- loop bodies
- final workflow assembly

This makes workflows harder to read and harder to reuse across examples or real projects.

## Goal

Add module loading to the DSL so workflow authors can split Starlark code across multiple files using standard Starlark `load(...)`.

The feature should make it possible to:

- move shared path helpers into one module
- move task constructors into another module
- keep the entry workflow file focused on composition
- reuse helpers across multiple workflows

## Recommendation

Support standard Starlark `load(...)` rather than inventing a custom include mechanism.

Example:

```python
load("lib/paths.star", "feature_paths")
load("lib/poem_tasks.star", "default_executor", "write_poem_task", "extend_poem_task", "review_poem_task")

name = param("name")
paths = feature_paths(name)

wf = workflow(
    id = "poem",
    default_executor = default_executor,
    steps = [
        write_poem_task(paths),
        repeat_until(
            id = "extend_until_ready",
            max_iters = 4,
            steps = [
                extend_poem_task(paths),
                review_poem_task(paths),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
```

This keeps the DSL aligned with Starlark itself and avoids creating a second module system on top of it.

## Scope

This feature is only about workflow authoring structure.

It does not add:

- runtime branching
- dynamic workflow generation after load
- remote module loading
- package registries
- versioned dependency management

## Module Model

### Entry File

The entry workflow file is resolved from the workflow ID passed to:

```sh
daiag run --workflow <workflow-id> --workflows-lib <dir>
```

The entry file must define top-level `wf`.

### Loaded Files

Loaded `.star` files are helper modules.
They may define:

- constants
- helper functions
- task constructors
- loop constructors

They should not define top-level `wf`.

Recommendation:

- reject loaded modules that define top-level `wf`

That avoids ambiguity about which file is the executable workflow entrypoint.

## Syntax

Use normal Starlark syntax:

```python
load("lib/tasks.star", "write_poem_task", "review_poem_task")
```

Supported import style should be the standard Starlark one:

- first argument: module path
- remaining arguments: exported symbol names

No custom forms such as `import`, `include`, or wildcard imports should be added.

## Path Resolution

### `load()` Paths

`load()` paths should resolve relative to the importing module.

Example:

- entry file: `examples/poem/workflows/poem/poem.star`
- loaded module: `load("../lib/tasks.star", "write_poem_task")`
- resolved file: `examples/poem/workflows/lib/tasks.star`

This is the least surprising behavior for Starlark users and makes modules relocatable.

### Boundary Rules

Loaded module paths should:

- be local filesystem paths only
- end in `.star`
- resolve to a canonical path under the workflows library root

Reject:

- URLs
- absolute paths outside the allowed root
- paths that escape the allowed root through `..`

### `template_file(...)` Interaction

Imports are much more useful if prompt-template paths are also stable once code is split into modules.

Recommended rule:

- `template_file(...)` paths should resolve relative to the module where they are declared, not relative to the entry workflow file

Example:

- module: `examples/poem/workflows/lib/tasks.star`
- prompt path: `template_file("../agents/poem-writer.md", vars = {...})`
- resolved prompt: `examples/poem/workflows/agents/poem-writer.md` if interpreted relative to the module

Without this rule, helper modules would need to know the entry file's directory layout, which defeats much of the value of `load()`.

This is the main behavior change that should accompany imports.

### Runtime Artifact Paths

Artifact paths and other runtime string expressions should keep their existing meaning.

They are workflow data, not Starlark module paths.
They should continue to resolve against workflow execution context, not module file location.

So:

- `load(...)` and `template_file(...)` are module-relative
- artifact paths remain runtime/workdir-relative unless absolute

## Predeclared DSL Builtins

Loaded modules should receive the same predeclared DSL builtins as the entry file:

- `workflow(...)`
- `task(...)`
- `repeat_until(...)`
- `artifact(...)`
- `path_ref(...)`
- `json_ref(...)`
- `loop_iter(...)`
- `template_file(...)`
- `param(...)`
- `format(...)`
- `eq(...)`

This allows helper modules to construct task values normally.

## `param(...)` Rule

Technically, `param(...)` could be available in loaded modules.
But that makes module dependencies less explicit because a helper file can silently depend on external CLI parameters.

Recommended rule:

- allow `param(...)` only in the entry workflow file

Loaded modules should instead accept values as function arguments.

Good:

```python
# lib/paths.star
def feature_paths(name):
    feature_dir = format("examples/poem/docs/features/{name}", name = name)
    return {
        "feature_dir": feature_dir,
        "spec_path": format("{dir}/spec.md", dir = feature_dir),
    }
```

```python
# poem.star
load("lib/paths.star", "feature_paths")

name = param("name")
paths = feature_paths(name)
```

This keeps required workflow inputs visible in the entry file.

## Supported Export Patterns

Loaded modules should be able to export:

- plain strings
- dicts and lists
- executor configs
- `task(...)` values
- `repeat_until(...)` values
- helper functions that build those values

Most reusable code should be written as helper functions rather than prebuilt task values.

Reason:

- task IDs must remain globally unique
- helper functions can accept IDs, paths, or executor configs as arguments

Preferred:

```python
def review_poem_task(paths):
    return task(
        id = "review_poem",
        executor = {"cli": "claude", "model": "sonnet"},
        prompt = template_file(
            "../agents/poem-reviewer.md",
            vars = {
                "POEM_PATH": path_ref("extend_poem", "poem"),
                "REVIEW_PATH": paths["review_path"],
            },
        ),
        artifacts = {
            "review": artifact(paths["review_path"]),
        },
        result_keys = ["outcome", "line_count", "review_path"],
    )
```

Less preferred, but still valid:

```python
default_executor = {"cli": "codex", "model": "gpt-5.4"}
```

## Caching And Identity

Modules should be loaded once per top-level workflow load, keyed by canonical absolute path.

Effects:

- repeated `load(...)` of the same file should return the same evaluated module
- modules should not be re-executed for each import site
- cycle detection becomes straightforward

## Cycle Handling

The loader should detect module cycles and fail with a clear error.

Example:

- `a.star` loads `b.star`
- `b.star` loads `a.star`

Recommended error shape:

```text
load cycle detected:
  poem.star
  lib/tasks.star
  lib/common.star
  lib/tasks.star
```

## Validation Rules

Reject these cases:

- entry file missing top-level `wf`
- loaded module defines top-level `wf`
- `load(...)` references a missing file
- `load(...)` references a path outside the allowed root
- `load(...)` imports a symbol the module does not export
- module cycle detected
- `param(...)` used from a loaded module, if the recommended restriction is adopted

Existing workflow validation should continue to run after module loading:

- duplicate step IDs
- missing prompt variables
- invalid `path_ref(...)`
- invalid `json_ref(...)`
- invalid `loop_iter(...)`

## Error Reporting

Errors should report both:

- the file where the failure happened
- the import chain that led there, when relevant

Example:

```text
load workflow: examples/poem/workflows/lib/tasks.star: missing exported symbol "review_poem_task"
import chain:
  examples/poem/workflows/poem/poem.star
  examples/poem/workflows/lib/tasks.star
```

Good import-chain errors matter because workflow authors will otherwise spend time debugging the wrong file.

## Example Split

One reasonable split for the poem example would be:

```text
examples/poem/workflows/
  poem.star
  lib/
    paths.star
    tasks.star
```

### `poem.star`

```python
load("lib/paths.star", "feature_paths")
load("lib/tasks.star", "default_executor", "write_poem_task", "extend_poem_task", "review_poem_task")

name = param("name")
paths = feature_paths(name)

wf = workflow(
    id = "poem",
    default_executor = default_executor,
    steps = [
        write_poem_task(paths),
        repeat_until(
            id = "extend_until_ready",
            max_iters = 4,
            steps = [
                extend_poem_task(paths),
                review_poem_task(paths),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
```

### `lib/paths.star`

```python
def feature_paths(name):
    feature_dir = format("examples/poem/docs/features/{name}", name = name)
    return {
        "feature_dir": feature_dir,
        "spec_path": format("{dir}/spec.md", dir = feature_dir),
        "poem_path": format("{dir}/poem.md", dir = feature_dir),
        "review_path": format(
            "{dir}/review-{iter}.txt",
            dir = feature_dir,
            iter = loop_iter("extend_until_ready"),
        ),
    }
```

### `lib/tasks.star`

```python
default_executor = {"cli": "codex", "model": "gpt-5.4"}

def write_poem_task(paths):
    return task(
        id = "write_poem",
        prompt = template_file(
            "../agents/poem-writer.md",
            vars = {
                "SPEC_PATH": paths["spec_path"],
                "POEM_PATH": paths["poem_path"],
            },
        ),
        artifacts = {
            "poem": artifact(paths["poem_path"]),
        },
        result_keys = ["topic", "line_count", "poem_path"],
    )
```

## Why This Feature Is Worth Adding

Benefits:

- cleaner workflow entry files
- easier reuse across multiple workflows
- less copy/paste of task definitions
- more natural Starlark authoring model

Costs:

- custom `Load` implementation in the loader
- module cache and cycle detection
- slightly more complex path semantics

The tradeoff is favorable once workflows move beyond toy size.

## Suggested Implementation Order

1. add loader support for standard Starlark `load(...)`
2. resolve module paths relative to the importing file
3. add module cache and cycle detection
4. restrict `load(...)` to allowed local roots
5. make `template_file(...)` module-relative
6. reject `wf` in loaded modules
7. optionally restrict `param(...)` to the entry file

## Recommendation Summary

Add `load(...)`.

Design it as standard Starlark module loading with:

- module-relative import resolution
- module-relative `template_file(...)`
- local-files-only boundaries
- no `wf` in helper modules
- entry-file ownership of `param(...)`

That gives the DSL a real composition mechanism without inventing a custom workflow-specific import feature.
