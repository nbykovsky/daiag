name = input("name")
feature_dir = format("examples/development-workflow/docs/features/{name}", name = name)
prd_path = format("{dir}/prd.md", dir = feature_dir)
spec_path = format("{dir}/spec.md", dir = feature_dir)

wf = workflow(
    id = "feature-development",
    inputs = ["name"],
    default_executor = {"cli": "claude", "model": "sonnet"},
    steps = [
        subworkflow(
            id = "spec_refinement",
            workflow = "spec-refinement.star",
            inputs = {
                "feature_dir": feature_dir,
                "prd_path": prd_path,
                "spec_path": spec_path,
            },
        ),
        task(
            id = "write_qa_tests",
            prompt = template_file(
                "../agents/qa-test-writer.md",
                vars = {
                    "SPEC_PATH": path_ref("spec_refinement", "spec"),
                    "QA_TESTS_PATH": format("{dir}/qa_tests.md", dir = feature_dir),
                    "STATUS_PATH": format("{dir}/qa_test_write_status.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "qa_tests": artifact(format("{dir}/qa_tests.md", dir = feature_dir)),
                "qa_test_write_status": artifact(format("{dir}/qa_test_write_status.md", dir = feature_dir)),
            },
            result_keys = ["outcome", "qa_tests_path", "status_path"],
        ),
        task(
            id = "split_spec_into_tasks",
            prompt = template_file(
                "../agents/spec-task-splitter.md",
                vars = {
                    "FEATURE_DIR": feature_dir,
                    "SPEC_PATH": path_ref("spec_refinement", "spec"),
                    "TASK_INDEX_PATH": format("{dir}/tasks.md", dir = feature_dir),
                    "STATUS_PATH": format("{dir}/task_split_status.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "task_index": artifact(format("{dir}/tasks.md", dir = feature_dir)),
                "task_split_status": artifact(format("{dir}/task_split_status.md", dir = feature_dir)),
            },
            result_keys = ["outcome", "task_index_path", "status_path"],
        ),
        task(
            id = "execute_tasks",
            executor = {"cli": "codex", "model": "gpt-5.4"},
            prompt = template_file(
                "../agents/task-batch-executor.md",
                vars = {
                    "TASK_INDEX_PATH": path_ref("split_spec_into_tasks", "task_index"),
                    "STATUS_PATH": format("{dir}/task_execution_status.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "task_index": artifact(path_ref("split_spec_into_tasks", "task_index")),
                "task_execution_status": artifact(format("{dir}/task_execution_status.md", dir = feature_dir)),
            },
            result_keys = ["outcome", "task_index_path", "status_path"],
        ),
        task(
            id = "refine_code",
            prompt = template_file(
                "../agents/code-refiner.md",
                vars = {
                    "FEATURE_DIR": feature_dir,
                    "SPEC_PATH": path_ref("spec_refinement", "spec"),
                    "STATUS_PATH": format("{dir}/code_review_status.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "code_review_status": artifact(format("{dir}/code_review_status.md", dir = feature_dir)),
            },
            result_keys = ["outcome", "status_path"],
        ),
        task(
            id = "run_qa_refiner",
            prompt = template_file(
                "../agents/qa-refiner.md",
                vars = {
                    "FEATURE_DIR": feature_dir,
                    "QA_TESTS_PATH": path_ref("write_qa_tests", "qa_tests"),
                    "STATUS_PATH": format("{dir}/qa_status.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "qa_status": artifact(format("{dir}/qa_status.md", dir = feature_dir)),
            },
            result_keys = ["outcome", "status_path"],
        ),
        task(
            id = "update_docs",
            prompt = template_file(
                "../agents/docs-updater.md",
                vars = {
                    "SPEC_PATH": path_ref("spec_refinement", "spec"),
                    "STATUS_PATH": format("{dir}/docs_update_status.md", dir = feature_dir),
                },
            ),
            artifacts = {
                "docs_update_status": artifact(format("{dir}/docs_update_status.md", dir = feature_dir)),
            },
            result_keys = ["outcome", "status_path"],
        ),
    ],
)
