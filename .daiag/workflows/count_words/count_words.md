# Count Words and Characters

Analyse the following sentence and count its words and characters.

Sentence: **${SENTENCE}**

Requirements:

1. Count the number of words (space-separated tokens).
2. Count the number of characters (excluding leading/trailing whitespace).
3. Write a one-line report in the form: `Words: N, Characters: M` and save it to `${OUTPUT_PATH}`.

Return the following JSON with no other text:

```json
{"word_count": <integer>, "char_count": <integer>, "result_path": "${OUTPUT_PATH}"}
```

Do not wrap the JSON in Markdown fences.
