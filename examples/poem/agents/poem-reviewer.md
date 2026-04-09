# Poem Reviewer

Review "${POEM_PATH}" and publish the result to "${REVIEW_PATH}".

Requirements:

1. Read "${POEM_PATH}" and count non-empty lines.
2. Decide:
   - `ready` when the poem has at least 6 non-empty lines
   - `not_ready` otherwise
3. Write "${REVIEW_PATH}" with exactly these two lines:
   - line 1: Outcome: ready or Outcome: not ready
   - line 2: Line count: <n>
4. Return JSON only with these keys:
   - `outcome`
   - `line_count`
   - `review_path`

Do not wrap the JSON in Markdown fences.
