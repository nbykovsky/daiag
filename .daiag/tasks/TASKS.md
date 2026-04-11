# Task Index

This file is the authoritative index of available tasks in `.daiag/tasks/`.

Update this file whenever a task pair is created or modified.
The `workflow-author` agent reads this file to discover available tasks and their contracts.

---

## write_topic_poem

Writes a poem on a given topic to a specified file.

Helper: `write_topic_poem_task(step_id, topic, line_count, poem_path)`

Inputs:
- `topic` — poem subject (string)
- `line_count` — number of lines to write (string)
- `poem_path` — path to write the poem to

Artifacts:
- `poem` → `poem_path`

Returns: `topic`, `line_count`, `poem_path`
