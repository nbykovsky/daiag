load("lib/paths.star", "feature_paths")
load("lib/tasks.star", "default_executor", "write_poem_task", "extend_poem_task", "review_poem_task")

name = param("name")
paths = feature_paths(name)

wf = workflow(
    id = "poem",
    default_executor = default_executor,
    steps = [
        write_poem_task(paths),
        repeat_until(
            id = "extend_until_ready",
            max_iters = 4,
            steps = [
                extend_poem_task(paths),
                review_poem_task(paths),
            ],
            until = eq(json_ref("review_poem", "outcome"), "ready"),
        ),
    ],
)
