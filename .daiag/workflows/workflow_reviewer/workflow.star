workflow_id = "workflow_reviewer"

workflow_id_input = input("workflow_id")

review_path = format("{run_dir}/workflow_reviewer/review.md", run_dir = run_dir())
workflows_dir = format("{projectdir}/.daiag/workflows/{workflow_id}", projectdir = projectdir(), workflow_id = workflow_id_input)

wf = workflow(
    id = workflow_id,
    inputs = ["workflow_id"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "review_workflow",
            prompt = template_file("review_workflow.md", vars = {
                "WORKFLOW_ID": workflow_id_input,
                "WORKFLOWS_DIR": workflows_dir,
                "REVIEW_PATH": review_path,
            }),
            artifacts = {"review": artifact(review_path)},
            result_keys = ["review_path", "outcome"],
        ),
    ],
    output_artifacts = {"review": path_ref("review_workflow", "review")},
    output_results = {
        "review_path": json_ref("review_workflow", "review_path"),
        "outcome": json_ref("review_workflow", "outcome"),
    },
)
