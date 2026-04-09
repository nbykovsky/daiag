name = param("name")
feature_dir = format("examples/poem/docs/features/{name}", name = name)

wf = workflow(
    id = "poem",
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file(
                "examples/poem/agents/poem-writer.md",
                vars = {
                    "SPEC_PATH": format("{dir}/spec.md", dir = feature_dir),
                    "POEM_PATH": format("{dir}/poem.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "poem": artifact(format("{dir}/poem.md", dir = feature_dir)),
            },
            result_keys = ["topic", "line_count", "poem_path"],
        ),
        repeat_until(
            id = "extend_until_ready",
            max_iters = 4,
            steps = [
                task(
                    id = "extend_poem",
                    prompt = template_file(
                        "examples/poem/agents/poem-extender.md",
                        vars = {
                            "POEM_PATH": path_ref("write_poem", "poem"),
                        },
                    ),
                    artifacts = {
                        "poem": artifact(path_ref("write_poem", "poem")),
                    },
                    result_keys = ["before_line_count", "after_line_count", "poem_path"],
                ),
                task(
                    id = "review_poem",
                    executor = {"cli": "claude", "model": "sonnet"},
                    prompt = template_file(
                        "examples/poem/agents/poem-reviewer.md",
                        vars = {
                            "POEM_PATH": path_ref("extend_poem", "poem"),
                            "REVIEW_PATH": format(
                                "{dir}/review-{iter}.txt",
                                dir = feature_dir,
                                iter = loop_iter("extend_until_ready"),
                            ),
                        },
                    ),
                    artifacts = {
                        "review": artifact(
                            format(
                                "{dir}/review-{iter}.txt",
                                dir = feature_dir,
                                iter = loop_iter("extend_until_ready"),
                            )
                        ),
                    },
                    result_keys = ["outcome", "line_count", "review_path"],
                ),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
