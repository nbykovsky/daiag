workflow_id = "workflow_bootstrapper"
description = input("description")
workflows_lib = input("workflows_lib")

blueprint_path = format("{run_dir}/workflow_bootstrapper/blueprint.md", run_dir = run_dir())
summary_path = format("{run_dir}/workflow_bootstrapper/summary.md", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["description", "workflows_lib"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "bootstrap",
            prompt = template_file("bootstrap.md", vars = {
                "DESCRIPTION": description,
                "WORKFLOWS_LIB": workflows_lib,
                "BLUEPRINT_PATH": blueprint_path,
                "SUMMARY_PATH": summary_path,
            }),
            artifacts = {
                "blueprint": artifact(blueprint_path),
                "summary": artifact(summary_path),
            },
            result_keys = ["workflow_id", "workflow_path", "outcome"],
        ),
    ],
    output_artifacts = {
        "blueprint": path_ref("bootstrap", "blueprint"),
        "summary": path_ref("bootstrap", "summary"),
    },
    output_results = {
        "workflow_id": json_ref("bootstrap", "workflow_id"),
        "workflow_path": json_ref("bootstrap", "workflow_path"),
        "outcome": json_ref("bootstrap", "outcome"),
    },
)
