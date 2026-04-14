workflow_id = "workflow_improver"

workflow_id_input = input("workflow_id")

review_path = format("{run_dir}/workflow_improver/review.md", run_dir = run_dir())
changes_path = format("{run_dir}/workflow_improver/changes.md", run_dir = run_dir())
workflow_star_path = format("{projectdir}/.daiag/workflows/{workflow_id}/workflow.star", projectdir = projectdir(), workflow_id = workflow_id_input)

wf = workflow(
    id = workflow_id,
    inputs = ["workflow_id"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        repeat_until(
            id = "improve_loop",
            max_iters = 3,
            steps = [
                subworkflow(
                    id = "review",
                    workflow = "workflow_reviewer",
                    inputs = {
                        "workflow_id": workflow_id_input,
                        "review_path": review_path,
                    },
                ),
                subworkflow(
                    id = "update",
                    workflow = "workflow_updater",
                    inputs = {
                        "workflow_id": workflow_id_input,
                        "review_path": review_path,
                        "changes_path": changes_path,
                    },
                ),
            ],
            until = eq(json_ref("update", "outcome"), "nothing_to_apply"),
        ),
    ],
    output_artifacts = {
        "review": review_path,
        "changes": changes_path,
        "workflow_star": workflow_star_path,
    },
    output_results = {},
)
