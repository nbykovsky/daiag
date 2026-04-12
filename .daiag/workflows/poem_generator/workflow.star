workflow_id = "poem_generator"
n = input("n")
poem_path = format("{workflow_id}/poem.md", workflow_id = workflow_id)

wf = workflow(
    id = workflow_id,
    inputs = ["n"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        task(
            id = "write_poem",
            prompt = template_file("write_poem.md", vars = {
                "N": n,
                "POEM_PATH": poem_path,
            }),
            artifacts = {"poem": artifact(poem_path)},
            result_keys = ["poem_path"],
        ),
    ],
    output_artifacts = {"poem": path_ref("write_poem", "poem")},
    output_results = {"poem_path": json_ref("write_poem", "poem_path")},
)
