claude_executor = {"cli": "claude", "model": "claude-haiku-4-5-20251001"}

def write_draft_task(topic, draft_path):
    return task(
        id = "write_draft",
        executor = claude_executor,
        prompt = template_file("../write-draft.md", vars = {
            "TOPIC": topic,
            "DRAFT_PATH": draft_path,
        }),
        artifacts = {"draft": artifact(draft_path)},
        result_keys = ["draft_path", "paragraph_count"],
    )

def improve_draft_task():
    return task(
        id = "improve_draft",
        executor = claude_executor,
        prompt = template_file("../improve-draft.md", vars = {
            "DRAFT_PATH": path_ref("write_draft", "draft"),
        }),
        artifacts = {"draft": artifact(path_ref("write_draft", "draft"))},
        result_keys = ["draft_path", "before_paragraph_count", "after_paragraph_count"],
    )

def review_draft_task(review_path):
    return task(
        id = "review_draft",
        executor = claude_executor,
        prompt = template_file("../review-draft.md", vars = {
            "DRAFT_PATH": path_ref("improve_draft", "draft"),
            "REVIEW_PATH": review_path,
        }),
        artifacts = {"review": artifact(review_path)},
        result_keys = ["outcome", "paragraph_count", "review_path"],
    )
