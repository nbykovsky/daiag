default_executor = {"cli": "codex", "model": "gpt-5.4"}

def write_poem_task(paths):
    return task(
        id = "write_poem",
        prompt = template_file(
            "../../agents/poem-writer.md",
            vars = {
                "SPEC_PATH": paths["spec_path"],
                "POEM_PATH": paths["poem_path"],
            },
        ),
        artifacts = {
            "poem": artifact(paths["poem_path"]),
        },
        result_keys = ["topic", "line_count", "poem_path"],
    )

def extend_poem_task(paths):
    return task(
        id = "extend_poem",
        prompt = template_file(
            "../../agents/poem-extender.md",
            vars = {
                "POEM_PATH": path_ref("write_poem", "poem"),
            },
        ),
        artifacts = {
            "poem": artifact(path_ref("write_poem", "poem")),
        },
        result_keys = ["before_line_count", "after_line_count", "poem_path"],
    )

def review_poem_task(paths):
    return task(
        id = "review_poem",
        executor = {"cli": "claude", "model": "sonnet"},
        prompt = template_file(
            "../../agents/poem-reviewer.md",
            vars = {
                "POEM_PATH": path_ref("extend_poem", "poem"),
                "REVIEW_PATH": paths["review_path"],
            },
        ),
        artifacts = {
            "review": artifact(paths["review_path"]),
        },
        result_keys = ["outcome", "line_count", "review_path"],
    )
