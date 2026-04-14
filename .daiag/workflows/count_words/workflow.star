text = input("text")
result_path = format("{run_dir}/count_words_step/result.json", run_dir = run_dir())

wf = workflow(
    id = "count_words",
    inputs = ["text"],
    default_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"},
    steps = [
        task(
            id = "count_words_step",
            prompt = template_file("count_words_step.md", vars = {
                "TEXT": text,
                "RESULT_PATH": result_path,
            }),
            artifacts = {"result": artifact(result_path)},
            result_keys = ["word_count", "original_text"],
        ),
    ],
    output_artifacts = {"result": path_ref("count_words_step", "result")},
    output_results = {
        "word_count": json_ref("count_words_step", "word_count"),
        "original_text": json_ref("count_words_step", "original_text"),
    },
)
