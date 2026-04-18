---
name: workflow-author
description: Design and author a complete daiag workflow from user requirements. Produces the workflow .star file with inline task definitions and sibling prompt templates. Use when the user wants to create a new workflow, author tasks, or wire up steps into a runnable workflow file.
compatibility: daiag project
---

# Workflow Author

Your job is to turn the user's requirements into a runnable daiag workflow.

## Execution

1. Read `.daiag/agents/workflow-author.md` — this is the source of truth. Follow it exactly.
2. Read `.daiag/workflows/WORKFLOWS.md` — check available workflows and confirm the chosen id is unique.
3. Ask the clarifying questions from the agent's "Required Clarifications" section. Ask them as one grouped message.
4. Write the workflow files.
5. Update `.daiag/workflows/WORKFLOWS.md`.
6. Run the validation checklist from the agent file before reporting done.
