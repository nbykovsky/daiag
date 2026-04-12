# Add Row

Inputs:
- `file_name`: ${FILE_NAME}

Instructions:
1. Read the file at `${FILE_NAME}`.
2. Examine all existing rows to identify the content style, format, and pattern (e.g. list items, CSV values, sentences, numbered lines).
3. Compose exactly one new row that matches the existing style and continues naturally from the last row.
4. Append the new row as the last line of the file. Do not insert blank lines before or after it.
5. Write the updated content back to `${FILE_NAME}`, preserving all existing rows exactly as they are.

Outputs:
- Write/update: ${FILE_NAME}
- Return JSON with keys:
  - `file_path`: set to `${FILE_NAME}`

Do not wrap the JSON in Markdown fences.
