workflow_id = "workflow_author_from_blueprint"
blueprint_path = input("blueprint_path")
summary_path = "workflow_author_from_blueprint/summary.md"

wf = workflow(
    id = workflow_id,
    inputs = ["blueprint_path"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "implement_from_blueprint",
            prompt = template_file("implement_from_blueprint.md", vars = {
                "BLUEPRINT_PATH": blueprint_path,
                "SUMMARY_PATH": summary_path,
            }),
            artifacts = {"summary": artifact(summary_path)},
            result_keys = ["workflow_path", "outcome"],
        ),
    ],
    output_artifacts = {
        "summary": path_ref("implement_from_blueprint", "summary"),
    },
    output_results = {
        "workflow_path": json_ref("implement_from_blueprint", "workflow_path"),
        "outcome": json_ref("implement_from_blueprint", "outcome"),
    },
)
