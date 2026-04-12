# Count Rows

Inputs:
- `file_name`: ${FILE_NAME}
- `m`: ${M}
- `status_path`: ${STATUS_PATH}

Instructions:
1. Read the file at `${FILE_NAME}`.
2. Count the total number of non-empty lines in the file.
3. If the line count is greater than `${M}`, set `outcome` to `done`; otherwise set it to `continue`.
4. Write a JSON object with `outcome` and `row_count` to `${STATUS_PATH}`, replacing any existing content.

Outputs:
- Write/update: ${STATUS_PATH}
- Return JSON with keys:
  - `outcome`: one of `done`, `continue`
  - `row_count`: the number of non-empty lines counted

Do not wrap the JSON in Markdown fences.
