workflow_id = "workflow_patcher"

report_path = input("report_path")
target_workflow_id = input("workflow_id")
workflows_lib = input("workflows_lib")

status_path = format("{run_dir}/workflow_patcher/status.md", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["report_path", "workflow_id", "workflows_lib"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "apply_review",
            prompt = template_file("apply_review.md", vars = {
                "REPORT_PATH": report_path,
                "WORKFLOW_ID": target_workflow_id,
                "WORKFLOWS_LIB": workflows_lib,
                "STATUS_PATH": status_path,
            }),
            artifacts = {"status": artifact(status_path)},
            result_keys = ["outcome"],
        ),
    ],
    output_artifacts = {},
    output_results = {
        "outcome": json_ref("apply_review", "outcome"),
    },
)
