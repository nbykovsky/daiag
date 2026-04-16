workflow_id = "workflow_assembler"
description = input("description")
workflows_lib = input("workflows_lib")

composition_plan_path = format("{run_dir}/workflow_assembler/composition_plan.md", run_dir = run_dir())
implement_summary_path = format("{run_dir}/workflow_assembler/implement_summary.md", run_dir = run_dir())

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
            result_keys = ["outcome"],
        ),
        task(
            id = "implement",
            prompt = template_file("implement.md", vars = {
                "WORKFLOWS_LIB": workflows_lib,
                "COMPOSITION_PLAN_PATH": path_ref("compose", "composition_plan"),
                "SUMMARY_PATH": implement_summary_path,
            }),
            artifacts = {"summary": artifact(implement_summary_path)},
            result_keys = ["workflow_id", "workflow_path", "outcome"],
        ),
        repeat_until(
            id = "review_loop",
            max_iters = 3,
            steps = [
                subworkflow(
                    id = "review",
                    workflow = "workflow_reviewer",
                    inputs = {
                        "workflow_id": json_ref("implement", "workflow_id"),
                        "workflows_lib": workflows_lib,
                        "report_name": format("review_{iter}.md", iter = loop_iter(loop_id = "review_loop")),
                    },
                ),
                subworkflow(
                    id = "patch",
                    workflow = "workflow_patcher",
                    inputs = {
                        "workflow_id": json_ref("implement", "workflow_id"),
                        "workflows_lib": workflows_lib,
                        "report_path": path_ref("review", "report"),
                    },
                ),
            ],
            until = eq(json_ref("patch", "outcome"), "complete"),
        ),
    ],
    output_artifacts = {
        "composition_plan": path_ref("compose", "composition_plan"),
        "last_report": path_ref("review", "report"),
    },
    output_results = {
        "workflow_id": json_ref("implement", "workflow_id"),
        "workflow_path": json_ref("implement", "workflow_path"),
        "outcome": json_ref("patch", "outcome"),
    },
)
