description = input("description")
workflows_lib = input("workflows_lib")

blueprint_path = format("{run_dir}/haiku_bootstrapper/blueprint.md", run_dir = run_dir())
summary_path = format("{run_dir}/haiku_bootstrapper/summary.md", run_dir = run_dir())

wf = workflow(
    id = "haiku_bootstrapper",
    inputs = ["description", "workflows_lib"],
    default_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"},
    steps = [
        task(
            id = "plan",
            prompt = template_file("plan.md", vars = {
                "DESCRIPTION": description,
                "WORKFLOWS_LIB": workflows_lib,
                "BLUEPRINT_PATH": blueprint_path,
            }),
            artifacts = {"blueprint": artifact(blueprint_path)},
            result_keys = ["blueprint_path", "outcome"],
        ),
        task(
            id = "author",
            prompt = template_file("author.md", vars = {
                "BLUEPRINT_PATH": path_ref("plan", "blueprint"),
                "WORKFLOWS_LIB": workflows_lib,
                "SUMMARY_PATH": summary_path,
            }),
            artifacts = {"summary": artifact(summary_path)},
            result_keys = ["workflow_id", "workflow_path", "outcome"],
        ),
    ],
    output_artifacts = {
        "blueprint": path_ref("plan", "blueprint"),
        "summary": path_ref("author", "summary"),
    },
    output_results = {
        "workflow_id": json_ref("author", "workflow_id"),
        "workflow_path": json_ref("author", "workflow_path"),
        "outcome": json_ref("author", "outcome"),
    },
)
