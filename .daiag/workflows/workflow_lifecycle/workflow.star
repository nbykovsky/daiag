workflow_id = "workflow_lifecycle"
description = input("description")
workflows_lib = format("{project}/.daiag/workflows", project = projectdir())

wf = workflow(
    id = workflow_id,
    inputs = ["description"],
    steps = [
        subworkflow(
            id = "bootstrap",
            workflow = "workflow_bootstrapper",
            inputs = {
                "description": description,
                "workflows_lib": workflows_lib,
            },
        ),
        repeat_until(
            id = "review_patch_loop",
            max_iters = 3,
            steps = [
                subworkflow(
                    id = "review",
                    workflow = "workflow_reviewer",
                    inputs = {
                        "workflow_id": json_ref("bootstrap", "workflow_id"),
                        "workflows_lib": workflows_lib,
                        "report_name": "review.md",
                    },
                ),
                subworkflow(
                    id = "patch",
                    workflow = "workflow_patcher",
                    inputs = {
                        "workflow_id": json_ref("bootstrap", "workflow_id"),
                        "workflows_lib": workflows_lib,
                        "report_path": path_ref("review", "report"),
                    },
                ),
            ],
            until = eq(json_ref("patch", "outcome"), "ready"),
        ),
    ],
    output_artifacts = {
        "last_report": path_ref("review", "report"),
    },
    output_results = {
        "workflow_id": json_ref("bootstrap", "workflow_id"),
        "workflow_path": json_ref("bootstrap", "workflow_path"),
        "outcome": json_ref("patch", "outcome"),
    },
)
