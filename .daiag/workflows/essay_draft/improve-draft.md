# Improve Essay Draft

Read the draft at `${DRAFT_PATH}` and append exactly one new paragraph to it.

Requirements:

1. Read `${DRAFT_PATH}` and count its existing paragraphs (non-empty blocks separated by blank lines).
2. Append one new paragraph that deepens or extends the essay's argument.
3. Separate the new paragraph from the last existing one with a blank line.
4. Preserve every existing paragraph exactly as written — do not edit them.
5. Save the updated draft back to `${DRAFT_PATH}`.

Return the following JSON with no other text:

```json
{"draft_path": "${DRAFT_PATH}", "before_paragraph_count": <integer>, "after_paragraph_count": <integer>}
```

Do not wrap the JSON in Markdown fences.
