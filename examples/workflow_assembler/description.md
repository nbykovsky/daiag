# workflow_assembler

## Goal

Create a workflow called `workflow_assembler` that assembles a complex daiag workflow
from existing catalog components. It takes a natural-language description and a
`workflows_lib` path, produces a detailed composition plan naming which catalog workflows
to use as subworkflow steps and how to wire them, then delegates implementation and
quality review to `workflow_lifecycle`.

## Inputs

- `description` — natural-language description of the workflow to build
- `workflows_lib` — absolute path to the workflow catalog

## Steps

### compose

A direct task that reads:
- `WORKFLOWS.md` from `workflows_lib` to discover available catalog workflows and their
  input/output contracts
- The `description` input

It writes a composition plan markdown artifact to `run_dir` and returns a single result key:

- `full_description` — a detailed structured description of the workflow to build, naming
  which existing catalog workflows to use as `subworkflow(...)` steps, their sequencing
  (linear, conditional with `when(...)`, or looping with `repeat_until(...)`), and the
  data flow between steps via `path_ref` and `json_ref`

### build

Calls `workflow_lifecycle` as a subworkflow with:
- `description` = `full_description` result from the `compose` step
- `workflows_lib` = the `workflows_lib` input

## Output Results

- `workflow_id` — forwarded from the `build` step
- `workflow_path` — forwarded from the `build` step
- `outcome` — forwarded from the `build` step

## References

- Catalog: `WORKFLOWS.md` in `workflows_lib`
- Building block used in `build` step: `workflow_lifecycle` — bootstraps a new workflow
  from a description then runs review and patch in a loop (max 3 iterations) until clean.
  Inputs: `description`, `workflows_lib`. Output results: `workflow_id`, `workflow_path`,
  `outcome`.
- Workflow language spec: `docs/workflow-language.md`
- CLI reference: `docs/workflow-cli.md`
