topic = input("topic")
poem_path = format("{run_dir}/hello_claude/poem.txt", run_dir = run_dir())

wf = workflow(
    id = "hello_claude",
    inputs = ["topic"],
    default_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file("write_poem.md", vars = {
                "TOPIC": topic,
                "POEM_PATH": poem_path,
            }),
            artifacts = {"poem": artifact(poem_path)},
            result_keys = ["poem_path"],
        ),
    ],
    output_artifacts = {"poem": path_ref("write_poem", "poem")},
    output_results = {"poem_path": json_ref("write_poem", "poem_path")},
)
