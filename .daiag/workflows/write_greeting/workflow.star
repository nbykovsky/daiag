greeting_name = input("greeting_name")
greeting_path = format("{run_dir}/write_greeting/greeting.txt", run_dir = run_dir())

wf = workflow(
    id = "write_greeting",
    inputs = ["greeting_name"],
    default_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"},
    steps = [
        task(
            id = "generate_greeting",
            prompt = template_file("generate_greeting.md", vars = {
                "GREETING_NAME": greeting_name,
                "GREETING_PATH": greeting_path,
            }),
            artifacts = {"greeting": artifact(greeting_path)},
            result_keys = ["message", "character_count"],
        ),
    ],
    output_artifacts = {"greeting_file": path_ref("generate_greeting", "greeting")},
    output_results = {
        "message": json_ref("generate_greeting", "message"),
        "character_count": json_ref("generate_greeting", "character_count"),
    },
)
