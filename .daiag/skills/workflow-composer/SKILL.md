---
name: workflow-composer
description: Convert high-level workflow requests into natural-language workflow blueprints. Read WORKFLOWS.md as a capability catalog, identify reusable workflows and missing capabilities, and produce a plain-language handoff. Do not author workflow code, inspect implementation files, or run checks.
compatibility: daiag project
---

# Workflow Composer

Your job is to turn a high-level workflow request into a natural-language workflow blueprint.

## Execution

1. Read `.daiag/agents/workflow-composer.md` — this is the source of truth. Follow it exactly.
2. Read `.daiag/workflows/WORKFLOWS.md` as the catalog of available workflow capabilities.
3. Do not read implementation references, generated workflow files, task prompt files, command documentation, implementation-agent instructions, or implementation source.
4. Produce a natural-language blueprint that identifies reusable existing workflows, missing capabilities, stage ordering, inputs, outputs, and open questions.
5. Do not create or modify files and do not run commands or checks.
