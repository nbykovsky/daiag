name = param("name")
feature_dir = format("examples/poem/docs/features/{name}", name = name)
default_executor = {"cli": "codex", "model": "gpt-5.4"}

spec_path = format("{dir}/spec.md", dir = feature_dir)
poem_path = format("{dir}/poem.md", dir = feature_dir)
review_path = format(
    "{dir}/review-{iter}.txt",
    dir = feature_dir,
    iter = loop_iter("extend_until_ready"),
)

write_poem = task(
    id = "write_poem",
    prompt = template_file(
        "examples/poem/agents/poem-writer.md",
        vars = {
            "SPEC_PATH": spec_path,
            "POEM_PATH": poem_path,
        },
    ),
    artifacts = {
        "poem": artifact(poem_path),
    },
    result_keys = ["topic", "line_count", "poem_path"],
)

extend_poem = task(
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
)

review_poem = task(
    id = "review_poem",
    executor = {"cli": "claude", "model": "sonnet"},
    prompt = template_file(
        "examples/poem/agents/poem-reviewer.md",
        vars = {
            "POEM_PATH": path_ref("extend_poem", "poem"),
            "REVIEW_PATH": review_path,
        },
    ),
    artifacts = {
        "review": artifact(review_path),
    },
    result_keys = ["outcome", "line_count", "review_path"],
)

wf = workflow(
    id = "poem",
    default_executor = default_executor,
    steps = [
        write_poem,
        repeat_until(
            id = "extend_until_ready",
            max_iters = 4,
            steps = [
                extend_poem,
                review_poem,
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
