output_path = "poems/output.md"

def write_poem_task(step_id, output_path):
    return task(
        id = step_id,
        prompt = template_file("simple_poem.md", vars = {
            "OUTPUT_PATH": output_path,
        }),
        artifacts = {"poem": artifact(output_path)},
        result_keys = ["poem_path"],
    )

wf = workflow(
    id = "simple_poem",
    inputs = [],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        write_poem_task("write_poem", output_path = output_path),
    ],
)
