load("../../../.daiag/tasks/write_topic_poem.star", "write_topic_poem_task")

topic = param("topic")
line_count = param("line_count")

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
