sentence = input("sentence")

wf = workflow(
    id = "count_words",
    inputs = ["sentence"],
    default_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"},
    steps = [
        task(
            id = "count",
            prompt = template_file("count_words.md", vars = {
                "SENTENCE": sentence,
                "OUTPUT_PATH": format("{workdir}/result.md", workdir = workdir()),
            }),
            artifacts = {"result": artifact(format("{workdir}/result.md", workdir = workdir()))},
            result_keys = ["word_count", "char_count", "result_path"],
        ),
    ],
)
