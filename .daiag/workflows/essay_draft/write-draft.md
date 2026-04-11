# Write Essay Draft

Write a 3-paragraph essay draft about the following topic: **${TOPIC}**

Requirements:

1. Write exactly 3 paragraphs separated by blank lines.
2. Each paragraph must be at least 2 sentences.
3. Keep the tone informative and clear.
4. Save the draft to `${DRAFT_PATH}`.

Return the following JSON with no other text:

```json
{"draft_path": "${DRAFT_PATH}", "paragraph_count": 3}
```

Do not wrap the JSON in Markdown fences.
