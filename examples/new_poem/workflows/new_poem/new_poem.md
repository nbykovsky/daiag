# Write Topic Poem

Write a poem about "${TOPIC}" to "${POEM_PATH}".

Requirements:

1. Write exactly ${LINE_COUNT} non-empty lines.
2. Replace "${POEM_PATH}" completely if it already exists.
3. Keep the poem as plain text with one line per row and no heading, numbering, or commentary.
4. Keep every line clearly about "${TOPIC}".
5. Return JSON only with these keys:
   - `topic`
   - `line_count`
   - `poem_path`

Do not wrap the JSON in Markdown fences.
