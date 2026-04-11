wf = workflow(
    id = "greet",
    default_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"},
    steps = [
        task(
            id = "write_greeting",
            prompt = template_file("greet.md", vars = {
                "OUTPUT_PATH": format("{workdir}/greeting.md", workdir = workdir()),
            }),
            artifacts = {"greeting": artifact(format("{workdir}/greeting.md", workdir = workdir()))},
            result_keys = ["greeting_path", "word_count"],
        ),
    ],
)
