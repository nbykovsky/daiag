# Workflow Authoring Pipeline Research

## Purpose

This document explores a workflow authoring pipeline for `daiag` where AI
agents build workflows from casual user descriptions by separating capability
discovery, primitive workflow creation, Starlark assembly, composition review,
and validation.

The goal is to make workflow authoring reliable for AI agents by keeping each
agent role narrow and by treating `.daiag/workflows/WORKFLOWS.md` as the public
catalog of reusable workflow capabilities.

## Problem

The current workflow authoring model mixes several different jobs:

- interpreting a casual user request
- deciding which workflow capabilities already exist
- identifying missing capabilities
- writing task prompt templates
- writing executable `task(...)` steps
- wiring control flow with `subworkflow(...)`, `repeat_until(...)`, and
  `when(...)`
- updating `.daiag/workflows/WORKFLOWS.md`

This makes a single authoring agent hard to guide. It must reason about both
prompt behavior and orchestration structure at the same time. The result is
confusing because a prompt-writing agent can accidentally invent workflow
composition, while a composition agent can accidentally invent new task
behavior that is not backed by a registered primitive workflow.

## Proposed Model

Split workflow authoring into five stages:

1. Gap analysis and structured planning
2. Primitive workflow authoring
3. Starlark workflow assembly
4. Composition review
5. Validation

Each stage has a specific input and output. Later stages should not recover
missing information by inspecting lower-level implementation files. They should
either use the published catalog contract or return a structured plan with
missing capabilities and open questions.

The expected control flow is:

1. Run gap analysis and structured planning.
2. If missing primitive workflows exist, author those primitives and rerun gap
   analysis against the original request and updated catalog.
3. When no gaps remain, assemble the composed Starlark workflow.
4. Review the assembled workflow against the original request and no-gap plan.
5. If the reviewer finds issues, return only to the Starlark workflow
   assembler.
6. When the reviewer approves, validate the workflow.

## Stage 1: Gap Analyzer and Structured Planner

The gap analyzer turns a casual user description into a structured composition
plan. The same artifact also records missing primitive capabilities and open
questions.

Inputs:

- the user's casual workflow description
- `.daiag/workflows/WORKFLOWS.md`

Allowed reads:

- `.daiag/workflows/WORKFLOWS.md`

Disallowed reads:

- workflow `.star` files
- task prompt `.md` files
- runtime source code
- implementation-agent instructions

Responsibilities:

- identify the user's overall goal
- identify starting runtime inputs
- identify desired final files and result values
- decompose the request into capability-level stages
- match stages to registered workflows when the catalog contract is sufficient
- define stage order and public artifact/result handoffs
- define loop bodies and exit conditions for `repeat_until(...)`
- define branch conditions for `when(...)`
- identify missing primitive capabilities
- identify open questions that block reliable planning

Output:

- a structured composition plan listing matched workflows, public handoffs,
  loops, branches, missing primitive workflows, and open questions

The gap analyzer should not write workflow code or prompt templates.

When the structured plan has no missing primitive workflows and no blocking open
questions, it is ready for the Starlark workflow assembler. There is no separate
composition-planner agent in this model.

## Stage 2: Primitive Workflow Author

The primitive workflow author creates missing capabilities found by the gap
analyzer.

Inputs:

- one missing primitive workflow brief from the structured plan
- `.daiag/workflows/WORKFLOWS.md`

Allowed writes:

- `.daiag/workflows/<workflow_id>/workflow.star`
- `.daiag/workflows/<workflow_id>/<task_id>.md`
- `.daiag/workflows/WORKFLOWS.md`

Responsibilities:

- create exactly one reusable primitive workflow per missing capability
- create exactly one `task(...)` step in that primitive workflow
- create exactly one sibling prompt template for the task
- declare explicit workflow inputs
- declare explicit `output_artifacts` and `output_results`
- document the new public contract in `WORKFLOWS.md`

The primitive workflow author should not compose multiple workflows. If a
missing capability is actually multi-stage, the gap analyzer should split it
into smaller primitive briefs or ask for clarification.

After this stage, the gap analyzer should rerun against the original user
request and the updated catalog. The loop continues until the structured plan
says there are no missing primitive capabilities and no blocking open questions.

## Stage 3: Starlark Workflow Assembler

The Starlark workflow assembler turns the no-gap structured plan into a
runnable orchestrating workflow.

Inputs:

- the no-gap structured composition plan
- `.daiag/workflows/WORKFLOWS.md`

Allowed writes:

- `.daiag/workflows/<workflow_id>/workflow.star`
- `.daiag/workflows/WORKFLOWS.md`

Disallowed writes:

- task prompt `.md` files

Responsibilities:

- write a workflow composed from registered workflows
- use `subworkflow(...)` for primitive stages
- use `repeat_until(...)` for iterative stage groups
- use `when(...)` for conditional stage groups
- bind every child workflow input explicitly
- expose final workflow outputs through `output_artifacts` and
  `output_results`
- update `WORKFLOWS.md` with the new composed workflow contract

The assembler should not use `task(...)` directly in composed workflows. If it
needs a new executable task, the request should go back to the gap analyzer and
primitive workflow author.

## Stage 4: Composition Reviewer

The composition reviewer checks whether the assembled workflow implements the
original request and the no-gap structured plan.

Inputs:

- the original user request
- the no-gap structured composition plan
- the generated composed workflow `.star` file
- the generated or updated `WORKFLOWS.md` entry for the composed workflow

Allowed reads:

- the generated composed workflow `.star` file
- `.daiag/workflows/WORKFLOWS.md`

Disallowed reads:

- primitive workflow `.star` files
- primitive task prompt `.md` files

Responsibilities:

- verify every requested stage is present
- verify every stage maps to the intended registered workflow
- verify child workflow inputs are bound from the right runtime input,
  constant, earlier artifact, or earlier result
- verify `repeat_until(...)` body, `max_iters`, and exit condition match the
  structured plan
- verify `when(...)` conditions and branches match the structured plan
- verify final `output_artifacts` and `output_results` expose the requested
  workflow contract
- verify the composed workflow does not introduce direct `task(...)` steps or
  prompt files
- verify the `WORKFLOWS.md` entry matches the assembled workflow contract

Output:

- `approved`, or
- a concise patch brief for the Starlark workflow assembler

Reviewer findings should loop back to the Starlark workflow assembler only. At
this point gaps are assumed to have been eliminated, so reviewer findings are
treated as assembly defects, not primitive capability defects. Reopening the
gap-analysis and primitive-authoring loop should be an explicit decision, not
the default reviewer path.

The review loop should have a maximum iteration count so the pipeline can stop
with a clear unresolved patch brief instead of cycling indefinitely.

## Stage 5: Validation

After the composition reviewer approves the workflow, run DSL validation.

Responsibilities:

- load and validate the composed workflow without executing tasks when a
  validate-only command is available
- treat validation failures as assembler defects and return them to the
  Starlark workflow assembler
- report the workflow ID, file path, inputs, output artifacts, and output
  results after validation passes

Validation checks syntax and DSL semantics. Composition review checks whether
the generated workflow still matches the original requirements. Both checks are
needed because they catch different failure classes.

## Workflow Catalog Contract

This model depends on `.daiag/workflows/WORKFLOWS.md` being more than an index.
It should behave like a public ABI for workflow composition.

Each workflow entry should document:

- workflow ID
- one-sentence purpose
- workflow file path
- runtime inputs
- output artifacts
- output results
- allowed values for enum-like outputs such as `outcome`
- whether each artifact is newly created or updates a provided input path
- important side effects
- whether outputs are intended to be used in loop or branch conditions

The current format already documents purpose, file path, inputs, output
artifacts, and output results. Composition will be more reliable if it also
documents enum values and mutation semantics.

Example extended entry shape:

```markdown
## file_row_grower

Repeatedly adds rows to a file in the existing content style until the line
count exceeds a threshold.

File: `.daiag/workflows/file_row_grower/workflow.star`

Inputs:
- `file_name` - path to the file to grow; updated in place
- `m` - row count threshold; loop exits when line count exceeds this value

Output Artifacts:
- `file` - the grown file at the path given by `file_name`
- `status` - `file_row_grower/count_status.json`

Output Results:
- `outcome` - one of `done`, `continue`
- `row_count` - final row count as an integer

Side Effects:
- Updates the file at `file_name` in place.

Composition Notes:
- `outcome` is suitable for `repeat_until(...)` exit checks.
- `file` is suitable as an input file for later subworkflows.
```

## Structured Plan Shape

The gap analyzer should produce structured natural language rather than
free-form prose. The assembler should be able to translate a no-gap plan
without guessing.

Suggested structured plan shape:

```markdown
Workflow goal:
- <one-sentence goal>

Runtime inputs:
- `<name>` - <description>

Final outputs:
- Artifact `<key>` - <description or source stage output>
- Result `<key>` - <description or source stage output>

Stages:
1. <stage name>
   - Workflow: `<workflow_id>`
   - Step ID: `<step_id>`
   - Inputs:
     - `<child_input>` <- <runtime input, constant, artifact, or result>
   - Uses outputs:
     - Artifact `<key>` for <later use or final output>
     - Result `<key>` for <later use, loop condition, branch condition, or
       final output>

Loops:
- `<loop_id>`
  - Body stages: `<step_id>`, `<step_id>`
  - Max iterations: <n>
  - Exit condition: `<step_id>.<result_key>` equals `<value>`

Branches:
- `<branch_id>`
  - Condition: `<step_id>.<result_key>` equals `<value>`
  - Then stages: `<step_id>`
  - Else stages: `<step_id>` or `none`

Catalog workflows used:
- `<workflow_id>`

Missing primitive workflows:
- none

If gaps exist, replace `none` with one or more entries:

- `<suggested_workflow_id>`
  - Purpose: <what the primitive task should do>
  - Inputs: <runtime inputs or files it should accept>
  - Output Artifacts: <files it should produce or update>
  - Output Results: <small JSON values it should return>
  - Acceptance: <how to tell the primitive succeeded>

Open questions:
- none
```

When `Missing primitive workflows` is `none` and `Open questions` is `none`,
the plan is ready for Starlark assembly.

## Review Brief Shape

When the composition reviewer does not approve the assembled workflow, it
should return a concise patch brief for the Starlark workflow assembler.

Suggested review brief shape:

```markdown
Review outcome:
- changes_requested

Findings:
1. <short issue title>
   - Requirement: <original requirement or structured plan item>
   - Current workflow: <what the assembled workflow does>
   - Required change: <what the assembler should change>

Allowed edit scope:
- `.daiag/workflows/<workflow_id>/workflow.star`
- `.daiag/workflows/WORKFLOWS.md`

Do not edit:
- primitive workflow `.star` files
- primitive prompt `.md` files
```

When the reviewer approves, the output can be just:

```markdown
Review outcome:
- approved
```

## Control Flow Notes

`repeat_until(...)` can contain `subworkflow(...)` stages, and a loop condition
can reference a result exposed by a subworkflow through `output_results`.
Therefore composed workflows can loop over primitive workflows without writing
new prompts in the composed workflow.

`when(...)` is useful for conditional side effects, but branch-internal steps
are not visible to later parent steps. A structured plan should avoid
requiring a later step to read "whichever branch ran" unless the workflow
language gains a join/select construct or the branch writes to a predetermined
artifact path that later stages can consume.

## Expected Benefits

- Narrower agents with clearer prompts
- Easier review of each handoff artifact
- Fewer accidental prompt inventions during composition
- Reusable primitive workflows with explicit contracts
- Composed workflows that are mostly wiring and control flow
- A review loop that fixes assembly mistakes without reopening primitive design
- A stronger `WORKFLOWS.md` catalog that can support automation

## Risks

- The process has more stages than a single authoring agent.
- `WORKFLOWS.md` must stay accurate or composition quality degrades.
- Missing capabilities must be scoped carefully so primitive workflows stay
  single-task.
- Some real workflows may need richer control-flow features than `when(...)`
  currently exposes.

## Migration Plan

1. Rename or replace the existing broad workflow authoring agents with narrower
   roles:
   - gap analyzer and structured planner
   - primitive workflow author
   - Starlark workflow assembler
   - composition reviewer
2. Update `.daiag/workflows/WORKFLOWS.md` entries to include enum values,
   mutation semantics, side effects, and composition notes.
3. Convert existing primitive workflows to the single-task contract where it
   fits.
4. Keep multi-stage workflows as composed workflows built from subworkflows.
5. Retire broad "author from blueprint" behavior or split it into primitive
   authoring, assembly, and review stages.

## Open Questions

- Should `WORKFLOWS.md` remain the only catalog format, or should a machine
  readable catalog be added later?
- Should composed workflows be forbidden from using `task(...)` by validation,
  or only by agent instruction?
- Should branch outputs become visible through a future join/select language
  feature?
- Should structured plans and review briefs have formal schemas to make
  handoffs easier to validate?
- What maximum review iteration count should the assembler/reviewer loop use?

## Related Documents

- `docs/workflow-bootstrap-cli.md`
