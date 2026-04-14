workflow_id = "workflow_bootstrapper"
description = input("description")
workflows_lib = input("workflows_lib")

blueprint_path = format("{run_dir}/workflow_bootstrapper/blueprint.md", run_dir = run_dir())
summary_path = format("{run_dir}/workflow_bootstrapper/summary.md", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["description", "workflows_lib"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "design",
            prompt = template_file("design.md", vars = {
                "DESCRIPTION": description,
                "WORKFLOWS_LIB": workflows_lib,
                "BLUEPRINT_PATH": blueprint_path,
            }),
            artifacts = {
                "blueprint": artifact(blueprint_path),
            },
            result_keys = ["workflow_id", "outcome"],
        ),
        # NOTE: If design returns needs_clarification, implement still executes — it reads
        # the blueprint, detects the signal, and returns needs_clarification itself.
        # This is a known v1 limitation (no conditional branching). Revisit if the DSL
        # gains conditional step execution.
        task(
            id = "implement",
            prompt = template_file("implement.md", vars = {
                "WORKFLOWS_LIB": workflows_lib,
                "BLUEPRINT_PATH": path_ref("design", "blueprint"),
                "SUMMARY_PATH": summary_path,
            }),
            artifacts = {
                "summary": artifact(summary_path),
                # The generated workflow.star path is dynamic (determined at runtime by the
                # design task's chosen workflow_id) and cannot be pre-declared as an artifact.
                # It is exposed to callers via the workflow_path result key instead.
            },
            result_keys = ["workflow_id", "workflow_path", "outcome"],
        ),
    ],
    output_artifacts = {
        "blueprint": path_ref("design", "blueprint"),
        "summary": path_ref("implement", "summary"),
    },
    output_results = {
        "workflow_id": json_ref("implement", "workflow_id"),
        "workflow_path": json_ref("implement", "workflow_path"),
        "outcome": json_ref("implement", "outcome"),
    },
)
