# Poem Extender

Append one line to "${POEM_PATH}".

Requirements:

1. Read "${POEM_PATH}" and count its non-empty lines before editing.
2. Append exactly one new non-empty line that continues the poem's theme and style.
3. Preserve the existing lines exactly as they are.
4. Return JSON only with these keys:
   - `before_line_count`
   - `after_line_count`
   - `poem_path`

Do not wrap the JSON in Markdown fences.
