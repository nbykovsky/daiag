topic = input("topic")

wf = workflow(
    id = "summarize",
    inputs = ["topic"],
    default_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"},
    steps = [
        task(
            id = "write_summary",
            prompt = template_file("summarize.md", vars = {
                "TOPIC": topic,
                "OUTPUT_PATH": format("{workdir}/summary.md", workdir = workdir()),
            }),
            artifacts = {"summary": artifact(format("{workdir}/summary.md", workdir = workdir()))},
            result_keys = ["summary_path", "sentence_count"],
        ),
    ],
)
