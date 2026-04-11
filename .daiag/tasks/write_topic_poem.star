def write_topic_poem_task(step_id, topic, line_count, poem_path):
    return task(
        id = step_id,
        executor = {"cli": "codex", "model": "gpt-5.4"},
        prompt = template_file(
            "write_topic_poem.md",
            vars = {
                "TOPIC": topic,
                "LINE_COUNT": line_count,
                "POEM_PATH": poem_path,
            },
        ),
        artifacts = {
            "poem": artifact(poem_path),
        },
        result_keys = [
            "topic",
            "line_count",
            "poem_path",
        ],
    )
