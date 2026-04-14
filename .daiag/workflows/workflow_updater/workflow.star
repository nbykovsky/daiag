workflow_id = "workflow_updater"
target_workflow_id = input("workflow_id")
review_path_input = input("review_path")

changes_path = format("{run_dir}/workflow_updater/changes.md", run_dir = run_dir())
workflow_star_path = format("{projectdir}/.daiag/workflows/{wf_id}/workflow.star", projectdir = projectdir(), wf_id = target_workflow_id)
workflows_dir = format("{projectdir}/.daiag/workflows/{wf_id}", projectdir = projectdir(), wf_id = target_workflow_id)

wf = workflow(
    id = workflow_id,
    inputs = ["workflow_id", "review_path"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "apply_review",
            prompt = template_file("apply_review.md", vars = {
                "REVIEW_PATH": review_path_input,
                "WORKFLOW_ID": target_workflow_id,
                "WORKFLOWS_DIR": workflows_dir,
                "CHANGES_PATH": changes_path,
            }),
            artifacts = {
                "workflow_star": artifact(workflow_star_path),
                "changes": artifact(changes_path),
            },
            result_keys = ["changes_path", "outcome"],
        ),
    ],
    output_artifacts = {
        "workflow_star": path_ref("apply_review", "workflow_star"),
        "changes": path_ref("apply_review", "changes"),
    },
    output_results = {
        "changes_path": json_ref("apply_review", "changes_path"),
        "outcome": json_ref("apply_review", "outcome"),
    },
)
