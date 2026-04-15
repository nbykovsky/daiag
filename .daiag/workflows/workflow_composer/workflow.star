workflow_id = "workflow_composer"
description = input("description")
workflows_lib = input("workflows_lib")

check_path = format("{run_dir}/{wf}/check_{iter}.md",
    run_dir = run_dir(), wf = workflow_id,
    iter = loop_iter(loop_id = "fill_catalog"))

wf = workflow(
    id = workflow_id,
    inputs = ["description", "workflows_lib"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        repeat_until(
            id = "fill_catalog",
            max_iters = 5,
            steps = [
                task(
                    id = "check",
                    prompt = template_file("check.md", vars = {
                        "DESCRIPTION": description,
                        "WORKFLOWS_LIB": workflows_lib,
                        "CHECK_PATH": check_path,
                    }),
                    artifacts = {"analysis": artifact(check_path)},
                    result_keys = ["outcome", "next_description"],
                ),
                when(
                    id = "create_if_needed",
                    condition = eq(json_ref("check", "outcome"), "create"),
                    steps = [
                        subworkflow(
                            id = "create_block",
                            workflow = "workflow_lifecycle",
                            inputs = {
                                "description": json_ref("check", "next_description"),
                                "workflows_lib": workflows_lib,
                            },
                        ),
                    ],
                ),
            ],
            until = eq(json_ref("check", "outcome"), "done"),
        ),
        subworkflow(
            id = "assemble",
            workflow = "workflow_bootstrapper",
            inputs = {
                "description": description,
                "workflows_lib": workflows_lib,
            },
        ),
    ],
    output_artifacts = {
        "last_check": path_ref("check", "analysis"),
    },
    output_results = {
        "workflow_id": json_ref("assemble", "workflow_id"),
        "workflow_path": json_ref("assemble", "workflow_path"),
        "outcome": json_ref("assemble", "outcome"),
    },
)
