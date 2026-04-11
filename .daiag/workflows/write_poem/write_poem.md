# Write Poem

Write a poem of exactly 10 lines to the file at `${POEM_PATH}`.

Requirements:
1. Write a poem of exactly 10 lines. Each line must be non-empty.
2. Create or overwrite the file at `${POEM_PATH}` with the poem text.
3. Do not add any metadata, headers, or extra blank lines — just the 10 lines of the poem.

Return only this JSON object:

{"poem_path": "<path where the poem was written>"}

Do not wrap the JSON in Markdown fences.
