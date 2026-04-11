poem_path = input("poem_path")

def write_poem_task(step_id, poem_path):
    return task(
        id = step_id,
        prompt = template_file("write_poem.md", vars = {
            "POEM_PATH": poem_path,
        }),
        artifacts = {"poem": artifact(poem_path)},
        result_keys = ["poem_path"],
    )

wf = workflow(
    id = "write_poem",
    inputs = ["poem_path"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        write_poem_task("write_poem_main", poem_path = poem_path),
    ],
    output_artifacts = {"poem": poem_path},
    output_results = {},
)
