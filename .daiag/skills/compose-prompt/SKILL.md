---
name: compose-prompt
description: Craft a precise natural-language description for `daiag bootstrap --workflow workflow_composer` and output the ready-to-run command. Use this skill whenever the user wants to create a new daiag workflow, describes what a workflow should do, asks to bootstrap a workflow, or says things like "I want a workflow that...", "create a workflow for...", "generate a workflow that...", or "run workflow_composer for me". Even if the user already provides some detail, invoke this skill to gather any missing clarifications and produce the correctly-formatted command.
compatibility: daiag project
---

# Compose Prompt

Your job is to help the user craft the `description` input that `daiag bootstrap --workflow workflow_composer` needs, then output the ready-to-run command.

The `workflow_composer` is fully automated — once you hand it a good description it will build the workflow without further questions. So the description must be a self-contained brief: precise enough that the composer can identify which catalog building blocks to reuse, detect missing capabilities, and assemble the final workflow on its own.

## Step 1 — Read the catalog

Read `.daiag/workflows/WORKFLOWS.md`.

You need this to know which building-block workflows already exist so you can name them in the description. The composer will match stage names against this catalog; if you name an existing workflow accurately it will reuse it. If a stage is new, your description must explain what that stage should do, read, write, and return.

## Step 2 — Gather requirements

Check what the user has already told you, then ask a single grouped message covering only the points that are still unclear:

1. **Goal** — what outcome should the complete workflow achieve? (one sentence)
2. **Starting inputs** — what values or file paths will the user provide at run time?
3. **Final outputs** — what files or returned values should the workflow produce?
4. **Stage ordering** — are there stages that must run in a specific order, or stages that depend on each other's outputs?
5. **New behavior** — for any stage that does not map to an existing catalog workflow, what should it read, write, and return? What does success look like?
6. **File updates** — should any stage update a user-provided file in place, or always create a new generated file?

Do not ask about implementation details: workflow syntax, executor choice, prompt format, file layout, or command flags.

## Step 3 — Produce the description

Write a plain-English description that covers all of the following in one coherent block:

- The end-to-end goal
- All starting inputs with their purpose
- The logical stages in order, each with:
  - Its purpose
  - Which existing catalog workflow to reuse (use the exact `workflow_id` from WORKFLOWS.md), or "new stage" if nothing fits
  - What it receives from earlier stages or user inputs
  - What it produces for later stages or as final output
- For every new stage: its inputs, outputs, and what "done" looks like
- The expected final outputs

Write it as if briefing a colleague who will implement the workflow without asking follow-up questions. Be specific about file paths only when the user specified them; otherwise use natural language like "the generated poem file".

**Good example:**

> Create a workflow that takes a GitHub PR URL and a reviewer persona as inputs. Stage 1: fetch the PR diff using the existing `github_pr_fetcher` workflow, which outputs the raw diff text. Stage 2: new stage — summarize the diff into a brief plain-language description of what changed (inputs: diff text; output: summary text file; done when the file exists and is non-empty). Stage 3: write a review comment in the style of the reviewer persona using the existing `code_review_pipeline` workflow with the summary as the file to review (inputs: summary file, persona string; outputs: review comment file and outcome). Final outputs: the review comment file and the outcome result.

**Weak example (avoid):**

> Create a workflow that reviews a PR. It should fetch the diff, summarize it, and write a review.

The weak version leaves ambiguity about which stages are new vs. reusable, what flows between stages, and what "done" means — the composer will have to guess.

## Step 4 — Output the command

Print two things:

**Description (for review):**
```
<the description text, exactly as you will pass it to --description>
```

**Command:**
```sh
daiag bootstrap \
  --workflow workflow_composer \
  --description "<description text on one line, with internal quotes escaped>"
```

Tell the user they can copy the command as-is. If they want to refine the description first, they can edit it and rerun.
