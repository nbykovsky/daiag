workflow_id = "code_review_pipeline"
file_path = input("file_path")
standards = input("standards")

violations_report_path = format("{run_dir}/code_review_pipeline/violations_report.md", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["file_path", "standards"],
    default_executor = {"cli": "claude", "model": "claude-sonnet-4-6"},
    steps = [
        task(
            id = "review",
            prompt = template_file("prompts/review.md", vars = {
                "FILE_PATH": file_path,
                "STANDARDS": standards,
                "VIOLATIONS_REPORT_PATH": violations_report_path,
            }),
            artifacts = {"violations_report": artifact(violations_report_path)},
            result_keys = ["outcome", "violation_count"],
        ),
        repeat_until(
            id = "fix_loop",
            max_iters = 3,
            steps = [
                task(
                    id = "fix",
                    prompt = template_file("prompts/fix.md", vars = {
                        "FILE_PATH": file_path,
                        "VIOLATIONS_REPORT_PATH": path_ref("review", "violations_report"),
                    }),
                    artifacts = {"file": artifact(file_path)},
                    result_keys = ["outcome"],
                ),
                task(
                    id = "re_review",
                    prompt = template_file("prompts/re_review.md", vars = {
                        "FILE_PATH": file_path,
                        "STANDARDS": standards,
                        "VIOLATIONS_REPORT_PATH": path_ref("review", "violations_report"),
                    }),
                    artifacts = {"violations_report": artifact(violations_report_path)},
                    result_keys = ["outcome", "violation_count"],
                ),
            ],
            until = eq(json_ref("re_review", "outcome"), "approved"),
        ),
    ],
    output_artifacts = {"violations_report": path_ref("re_review", "violations_report")},
    output_results = {
        "outcome": json_ref("re_review", "outcome"),
        "violation_count": json_ref("re_review", "violation_count"),
        "initial_outcome": json_ref("review", "outcome"),
        "initial_violation_count": json_ref("review", "violation_count"),
    },
)
