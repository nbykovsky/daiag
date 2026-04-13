workflow_id = "workflow_composer"
description = input("description")
workflows_lib = input("workflows_lib")
blueprint_path = format("{run_dir}/workflow_composer/blueprint.md", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["description", "workflows_lib"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "compose_workflow",
            prompt = template_file("compose_workflow.md", vars = {
                "DESCRIPTION": description,
                "WORKFLOWS_LIB": workflows_lib,
                "BLUEPRINT_PATH": blueprint_path,
            }),
            artifacts = {"blueprint": artifact(blueprint_path)},
            result_keys = ["blueprint_path", "outcome"],
        ),
    ],
    output_artifacts = {
        "blueprint": path_ref("compose_workflow", "blueprint"),
    },
    output_results = {
        "outcome": json_ref("compose_workflow", "outcome"),
        "blueprint_path": json_ref("compose_workflow", "blueprint_path"),
    },
)
