---
name: workflow-task-author
description: Create daiag workflow tasks as paired Starlark (.star) and Markdown (.md) files following project conventions. Use this skill whenever the user wants to create a new workflow task, define a task step, or author a daiag task pair. The skill loads the workflow conventions guide, asks clarifying questions about the task's purpose and requirements, then generates both files ready to use in the workflow.
compatibility: daiag project
---

# Workflow Task Author

Your job is to turn a requested workflow step into a pair of files:
- `.daiag/tasks/<step_name>.star` (Starlark definition)
- `.daiag/tasks/<step_name>.md` (Markdown prompt template)

Both files follow the conventions documented in [`.daiag/agents/workflow-task-author.md`](.daiag/agents/workflow-task-author.md).

## Workflow

1. **Understand the task intent** - Ask the user what step they want to create (if not already clear), and what it should accomplish
2. **Load the guide** - Read `.daiag/agents/workflow-task-author.md` to internalize the conventions
3. **Ask clarifying questions** - Before writing files, clarify:
   - Step name (the base name used for both files)
   - What files the task reads and what it writes
   - The JSON fields it should return
   - Whether it uses the default Codex executor or needs something different
   - Any other task-specific requirements
4. **Generate both files** - Create the `.star` and `.md` pair following the exact conventions from the guide
5. **Validate against the checklist** - Before finishing, verify all items in the validation checklist from the guide

## Key Conventions

- **File naming**: Use the same unsuffixed base name for both files and the exported helper function
- **Helper signature**: Export `def <step_name>_task(suffix, ...)` that accepts `suffix` as the first argument
- **Task ID**: Build it as `"<step_name>_" + suffix` to support task instance tracking
- **Prompt template**: Always use `template_file("<step_name>.md", vars = {...})` — never inline prompt text
- **Executor**: Default to Codex (`{"cli": "codex", "model": "gpt-5.4"}`) unless the user specifies otherwise
- **Artifacts**: Every output file must be wrapped in `artifact(...)`
- **Result keys**: Must match JSON fields returned by the agent exactly

## Prompt Template Structure

Use this structure:
- Short title
- One short opening instruction sentence
- Numbered requirements section
- JSON-only return contract (with allowed values for enums)
- End with: "Do not wrap the JSON in Markdown fences."

Optional sections:
- `Inputs:` (when multiple paths need disambiguation)
- `Outputs:` (when the task writes multiple files or both content and status)

## What You Generate

Before finishing, the user should see:
1. The complete `.star` file (ready to copy/paste)
2. The complete `.md` file (ready to copy/paste)
3. A summary of what the task does, what it reads/writes, and what it returns

The user can then save these to `.daiag/tasks/<step_name>.star` and `.daiag/tasks/<step_name>.md` respectively.

## Don't Guess

If the user hasn't specified:
- The step name
- What files the task reads/writes
- Required JSON output fields
- Executor requirements

Ask one focused question to clarify. Don't make assumptions — these details shape the entire task definition.
