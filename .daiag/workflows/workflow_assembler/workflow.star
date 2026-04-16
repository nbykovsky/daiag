workflow_id = "workflow_assembler"
description = input("description")
workflows_lib = input("workflows_lib")

composition_plan_path = format("{run_dir}/workflow_assembler/composition_plan.md", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["description", "workflows_lib"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "compose",
            prompt = template_file("compose.md", vars = {
                "DESCRIPTION": description,
                "WORKFLOWS_LIB": workflows_lib,
                "COMPOSITION_PLAN_PATH": composition_plan_path,
            }),
            artifacts = {"composition_plan": artifact(composition_plan_path)},
            result_keys = ["full_description"],
        ),
        subworkflow(
            id = "build",
            workflow = "workflow_lifecycle",
            inputs = {
                "description": json_ref("compose", "full_description"),
                "workflows_lib": workflows_lib,
            },
        ),
    ],
    output_artifacts = {
        "composition_plan": path_ref("compose", "composition_plan"),
    },
    output_results = {
        "workflow_id": json_ref("build", "workflow_id"),
        "workflow_path": json_ref("build", "workflow_path"),
        "outcome": json_ref("build", "outcome"),
    },
)
