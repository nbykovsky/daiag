# Review Essay Draft

Review the draft at `${DRAFT_PATH}` and write a review to `${REVIEW_PATH}`.

Requirements:

1. Read `${DRAFT_PATH}` and count its paragraphs (non-empty blocks separated by blank lines).
2. Decide the outcome:
   - `ready` when the draft has 5 or more paragraphs
   - `needs_work` when the draft has fewer than 5 paragraphs
3. Write `${REVIEW_PATH}` with exactly these two lines:
   - line 1: `Outcome: ready` or `Outcome: needs work`
   - line 2: `Paragraph count: <n>`

Return the following JSON with no other text:

```json
{"outcome": "<ready or needs_work>", "paragraph_count": <integer>, "review_path": "${REVIEW_PATH}"}
```

Do not wrap the JSON in Markdown fences.
