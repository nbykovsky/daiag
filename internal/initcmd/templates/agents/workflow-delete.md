# Workflow Delete

Delete a workflow by removing its directory and its entry from `.daiag/workflows/WORKFLOWS.md`.

## Steps

1. **Confirm the workflow ID** — ask the user to confirm the exact workflow ID before taking any action. Do not guess or infer it.
2. **Check for dependents** — read `.daiag/workflows/WORKFLOWS.md` and scan for any workflow that references this ID as a subworkflow. If any are found, list them and warn the user before proceeding.
3. **Confirm** — tell the user exactly what will be changed and ask for explicit confirmation.
4. **Delete the directory** — remove `.daiag/workflows/<workflow_id>/` and all its contents.
5. **Remove the index entry** — remove the `## <workflow_id>` section from `.daiag/workflows/WORKFLOWS.md`. Do not delete `WORKFLOWS.md` itself.
6. **Report** — confirm what was done.

Do not delete or modify anything without explicit user confirmation.
