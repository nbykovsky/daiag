workflow_id = "ensure_code_standards"

analysis_path = format("{run_dir}/ensure_code_standards/analysis.md", run_dir = run_dir())
standards_path = format("{project_dir}/docs/code-standards.md", project_dir = projectdir())

wf = workflow(
    id = workflow_id,
    inputs = [],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "analyze",
            prompt = template_file("analyze.md", vars = {
                "PROJECT_DIR": projectdir(),
                "ANALYSIS_PATH": analysis_path,
            }),
            artifacts = {"analysis": artifact(analysis_path)},
            result_keys = ["action"],
        ),
        when(
            id = "conditional_write",
            condition = neq(json_ref("analyze", "action"), "ok"),
            steps = [
                task(
                    id = "write",
                    prompt = template_file("write.md", vars = {
                        "ANALYSIS_PATH": path_ref("analyze", "analysis"),
                        "PROJECT_DIR": projectdir(),
                        "STANDARDS_PATH": standards_path,
                    }),
                    artifacts = {"standards": artifact(standards_path)},
                    result_keys = ["outcome"],
                ),
            ],
            else_steps = [],
        ),
    ],
    output_artifacts = {
        "analysis": path_ref("analyze", "analysis"),
        # absent when action == "ok" (write task skipped)
        "standards": path_ref("write", "standards"),
    },
    output_results = {
        "action": json_ref("analyze", "action"),
        # absent when action == "ok" (write task skipped)
        "outcome": json_ref("write", "outcome"),
    },
)
