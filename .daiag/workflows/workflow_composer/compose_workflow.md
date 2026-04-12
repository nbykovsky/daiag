# Compose Workflow Blueprint

Inputs:
- `description`: ${DESCRIPTION}
- `blueprint_path`: ${BLUEPRINT_PATH}

Instructions:

You are a requirements and composition planner. Your job is to turn a high-level
workflow description into a natural-language workflow blueprint. You do not know
the workflow implementation language, do not write workflow files, and do not run
commands or checks.

1. Read `.daiag/workflows/WORKFLOWS.md` to learn the available workflow catalog.
2. Summarize the available workflow catalog in your own words.
3. Decompose the description provided in `description` into natural-language stages.
4. Match each stage to an existing catalog workflow when the catalog description,
   inputs, and outputs fit the stage. Apply the reuse criteria below.
5. Mark unmatched stages as missing capabilities.
6. Define how information flows between stages using plain language:
   - parent/user-provided input
   - file produced by an earlier stage
   - value reported by an earlier stage
   - constant value from the request
7. Write the completed blueprint to `${BLUEPRINT_PATH}` using the exact format
   described in the Output Format section below.

## Reuse Criteria

Reuse an existing workflow when the catalog shows that:
- its purpose satisfies the stage
- the requested workflow can provide all of its required inputs
- it produces the files or values needed by later stages
- its side effects match the user request

Do not reuse an existing workflow when:
- it only partially satisfies the stage
- its required input cannot be provided by the requested workflow
- it does not produce the files or values needed later
- its side effects are unclear or risky

When reuse is uncertain, label it as uncertain and state the reason.

## Output Format

Write `${BLUEPRINT_PATH}` with exactly this content structure:

---

Workflow goal:
- <one-sentence outcome>

Starting inputs:
- `<name>` — <what the user provides>

Final outputs:
- `<name>` — <file or value the complete workflow should produce>

Stages:
1. <stage name>
   - Decision: reuse existing workflow `<workflow_id>` | create missing workflow | uncertain
   - Purpose: <what this stage does>
   - Receives: <user input, constant, or output from earlier stage>
   - Produces: <files and values needed by later stages or final output>
   - Notes: <assumptions or risks>

Information flow:
- <plain-language handoff from one stage to another>

Missing workflow briefs:
- `<suggested_workflow_id>`:
  - Purpose: <what it should do>
  - Inputs: <natural-language inputs>
  - Outputs: <natural-language files and values>
  - Acceptance: <how to tell this stage succeeded>

Open questions:
- <only unresolved questions that block a reliable plan>

---

If there are no missing workflows or open questions, write `none` for those sections.

Outputs:
- Write/update: ${BLUEPRINT_PATH}
- Return JSON with keys:
  - `blueprint_path`: set to `${BLUEPRINT_PATH}`
  - `outcome`: one of `complete`, `needs_clarification`

Do not wrap the JSON in Markdown fences.
