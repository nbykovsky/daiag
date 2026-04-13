workflow_id = "workflow_bootstrapper"
description = input("description")
workflows_lib = input("workflows_lib")

wf = workflow(
    id = workflow_id,
    inputs = ["description", "workflows_lib"],
    steps = [
        subworkflow(
            id = "compose_workflow",
            workflow = "workflow_composer",
            inputs = {
                "description": description,
                "workflows_lib": workflows_lib,
            },
        ),
        subworkflow(
            id = "author_workflow",
            workflow = "workflow_author_from_blueprint",
            inputs = {
                "blueprint_path": path_ref("compose_workflow", "blueprint"),
                "workflows_lib": workflows_lib,
            },
        ),
    ],
    output_artifacts = {
        "blueprint": path_ref("compose_workflow", "blueprint"),
        "summary": path_ref("author_workflow", "summary"),
    },
    output_results = {
        "workflow_id": json_ref("author_workflow", "workflow_id"),
        "workflow_path": json_ref("author_workflow", "workflow_path"),
        "outcome": json_ref("author_workflow", "outcome"),
    },
)
