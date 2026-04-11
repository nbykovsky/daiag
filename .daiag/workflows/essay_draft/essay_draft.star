load("lib/tasks.star", "write_draft_task", "improve_draft_task", "review_draft_task")

topic = input("topic")
draft_path = format("{workdir}/draft.md", workdir = workdir())
review_path = format(
    "{workdir}/review-{iter}.md",
    workdir = workdir(),
    iter = loop_iter("refine_until_ready"),
)

wf = workflow(
    id = "essay_draft",
    inputs = ["topic"],
    steps = [
        write_draft_task(topic, draft_path),
        repeat_until(
            id = "refine_until_ready",
            max_iters = 3,
            steps = [
                improve_draft_task(),
                review_draft_task(review_path),
            ],
            until = eq(json_ref("review_draft", "outcome"), "ready"),
        ),
    ],
)
