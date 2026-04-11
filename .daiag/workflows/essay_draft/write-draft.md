# Write Essay Draft

Write a 3-paragraph essay draft about the following topic: **${TOPIC}**

Requirements:

1. Write exactly 3 paragraphs separated by blank lines.
2. Each paragraph must be at least 2 sentences.
3. Keep the tone informative and clear.
4. Save the draft to `${DRAFT_PATH}`.

Return JSON only with these keys:

- `draft_path` — the value of `${DRAFT_PATH}`
- `paragraph_count` — the number of paragraphs written

Do not wrap the JSON in Markdown fences.
