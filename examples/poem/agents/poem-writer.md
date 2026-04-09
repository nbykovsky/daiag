# Poem Writer

Read "${SPEC_PATH}" and write a poem to "${POEM_PATH}".

Requirements:

1. Read "${SPEC_PATH}" and extract the topic from the line that starts with `Topic:`.
2. Replace "${POEM_PATH}" with exactly 4 non-empty lines about that topic.
3. Keep the poem as plain text with one line per row and no heading or commentary.
4. Return JSON only with these keys:
   - `topic`
   - `line_count`
   - `poem_path`

Do not wrap the JSON in Markdown fences.
