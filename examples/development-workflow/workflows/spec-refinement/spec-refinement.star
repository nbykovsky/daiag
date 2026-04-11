feature_dir = input("feature_dir")
prd_path = input("prd_path")
spec_path = input("spec_path")

wf = workflow(
    id = "spec-refinement",
    inputs = ["feature_dir", "prd_path", "spec_path"],
    default_executor = {"cli": "claude", "model": "sonnet"},
    steps = [
        task(
            id = "write_spec",
            prompt = template_file(
                "../../agents/spec-writer.md",
                vars = {
                    "FEATURE_DIR": feature_dir,
                    "PRD_PATH": prd_path,
                    "ARCH_PATH": "docs/architecture.md",
                    "SPEC_PATH": spec_path,
                    "STATUS_PATH": format("{dir}/spec_write_status.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "spec": artifact(spec_path),
                "spec_write_status": artifact(format("{dir}/spec_write_status.md", dir = feature_dir)),
            },
            result_keys = ["outcome", "spec_path", "status_path"],
        ),
        repeat_until(
            id = "refine_spec",
            max_iters = 3,
            steps = [
                task(
                    id = "review_spec",
                    prompt = template_file(
                        "../../agents/requirements-reviewer.md",
                        vars = {
                            "SPEC_PATH": path_ref("write_spec", "spec"),
                            "REVIEW_PATH": format(
                                "{dir}/spec_review_{iter}.md",
                                dir = feature_dir,
                                iter = loop_iter("refine_spec"),
                            ),
                        },
                    ),
                    artifacts = {
                        "spec": artifact(path_ref("write_spec", "spec")),
                        "review_report": artifact(
                            format(
                                "{dir}/spec_review_{iter}.md",
                                dir = feature_dir,
                                iter = loop_iter("refine_spec"),
                            )
                        ),
                    },
                    result_keys = ["outcome", "spec_path", "review_path"],
                ),
                task(
                    id = "address_review",
                    prompt = template_file(
                        "../../agents/review-addresser.md",
                        vars = {
                            "SPEC_PATH": path_ref("review_spec", "spec"),
                            "REVIEW_PATH": path_ref("review_spec", "review_report"),
                            "STATUS_PATH": format(
                                "{dir}/spec_refine_status_{iter}.md",
                                dir = feature_dir,
                                iter = loop_iter("refine_spec"),
                            ),
                        },
                    ),
                    artifacts = {
                        "spec": artifact(path_ref("review_spec", "spec")),
                        "spec_refine_status": artifact(
                            format(
                                "{dir}/spec_refine_status_{iter}.md",
                                dir = feature_dir,
                                iter = loop_iter("refine_spec"),
                            )
                        ),
                    },
                    result_keys = ["loop_outcome", "spec_path", "status_path"],
                ),
            ],
            until = eq(json_ref("address_review", "loop_outcome"), "stop"),
        ),
    ],
    output_artifacts = {
        "spec": path_ref("address_review", "spec"),
        "last_review": path_ref("review_spec", "review_report"),
    },
    output_results = {
        "outcome": json_ref("address_review", "loop_outcome"),
    },
)
