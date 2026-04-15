workflow_id = "workflow_reviewer"
target_workflow_id = input("workflow_id")
workflows_lib = input("workflows_lib")
report_name = input("report_name")

report_path = format("{run_dir}/workflow_reviewer/{report_name}", run_dir = run_dir(), report_name = report_name)

wf = workflow(
    id = workflow_id,
    inputs = ["workflow_id", "workflows_lib", "report_name"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "review_workflow",
            prompt = template_file("review_workflow.md", vars = {
                "WORKFLOW_ID": target_workflow_id,
                "WORKFLOWS_LIB": workflows_lib,
                "REPORT_PATH": report_path,
            }),
            artifacts = {
                "report": artifact(report_path),
            },
            result_keys = ["report_path"],
        ),
    ],
    output_artifacts = {
        "report": path_ref("review_workflow", "report"),
    },
    output_results = {
        "report_path": json_ref("review_workflow", "report_path"),
    },
)
