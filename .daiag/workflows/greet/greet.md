# Write a Greeting

Write a friendly three-sentence greeting message and save it to `${OUTPUT_PATH}`.

Requirements:

1. The greeting must be exactly three sentences.
2. It must be warm and welcoming.
3. Save the greeting to `${OUTPUT_PATH}`.

Return the following JSON with no other text:

```json
{"greeting_path": "${OUTPUT_PATH}", "word_count": <integer word count of the greeting>}
```

Do not wrap the JSON in Markdown fences.
