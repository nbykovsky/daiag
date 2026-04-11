# Write a Summary

Write a concise five-sentence summary about the following topic: **${TOPIC}**

Requirements:

1. Write exactly five sentences.
2. Each sentence should cover a distinct aspect of the topic.
3. Save the summary to `${OUTPUT_PATH}`.

Return the following JSON with no other text:

```json
{"summary_path": "${OUTPUT_PATH}", "sentence_count": <integer number of sentences written>}
```

Do not wrap the JSON in Markdown fences.
