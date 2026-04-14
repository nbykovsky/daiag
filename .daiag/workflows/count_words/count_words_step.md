Count the number of words in the following text by splitting on whitespace:

${TEXT}

Write a JSON file to: ${RESULT_PATH}

The JSON must contain:
- word_count: the total number of words in the text
- original_text: the original input text that was analyzed

Make sure the parent directory exists before writing (create it if needed).

Return ONLY a JSON object (no other text before or after):
{"word_count": <number>, "original_text": "${TEXT}"}
