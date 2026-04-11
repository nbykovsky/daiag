topic = param("topic")
line_count = param("line_count")

def write_topic_poem_task(step_id, topic, line_count, poem_path):
    return task(
        id = step_id,
        executor = {"cli": "codex", "model": "gpt-5.4"},
        prompt = template_file("new_poem.md", vars = {
            "TOPIC": topic,
            "LINE_COUNT": line_count,
            "POEM_PATH": poem_path,
        }),
        artifacts = {
            "poem": artifact(poem_path),
        },
        result_keys = [
            "topic",
            "line_count",
            "poem_path",
        ],
    )

wf = workflow(
    id = "new_poem",
    steps = [
        write_topic_poem_task(
            "main",
            topic = topic,
            line_count = line_count,
            poem_path = "examples/new_poem/poem.md",
        ),
    ],
)
