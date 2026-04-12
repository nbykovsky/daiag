workflow_id = "workflow_composer"
description = input("description")
blueprint_path = "workflow_composer/blueprint.md"

wf = workflow(
    id = workflow_id,
    inputs = ["description"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "compose_workflow",
            prompt = template_file("compose_workflow.md", vars = {
                "DESCRIPTION": description,
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
