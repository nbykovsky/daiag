workflow_id = "file_row_grower"
file_name = input("file_name")
file_path = format("{project}/{file_name}", project = projectdir(), file_name = file_name)
m = input("m")
status_path = format("{run_dir}/file_row_grower/count_status.json", run_dir = run_dir())

wf = workflow(
    id = workflow_id,
    inputs = ["file_name", "m"],
    default_executor = {"cli": "codex", "model": "gpt-5.4"},
    steps = [
        repeat_until(
            id = "grow_loop",
            max_iters = 15,
            steps = [
                task(
                    id = "add_row",
                    prompt = template_file("add_row.md", vars = {
                        "FILE_NAME": file_path,
                    }),
                    artifacts = {"file": artifact(file_path)},
                    result_keys = ["file_path"],
                ),
                task(
                    id = "count_rows",
                    prompt = template_file("count_rows.md", vars = {
                        "FILE_NAME": path_ref("add_row", "file"),
                        "M": m,
                        "STATUS_PATH": status_path,
                    }),
                    artifacts = {"status": artifact(status_path)},
                    result_keys = ["outcome", "row_count"],
                ),
            ],
            until = eq(json_ref("count_rows", "outcome"), "done"),
        ),
    ],
    output_artifacts = {
        "file": path_ref("add_row", "file"),
        "status": path_ref("count_rows", "status"),
    },
    output_results = {
        "outcome": json_ref("count_rows", "outcome"),
        "row_count": json_ref("count_rows", "row_count"),
    },
)
