Create a workflow called `code_review_pipeline`.

Inputs:
- `file_path` — absolute path to the source file to review
- `standards` — natural-language description of the coding standards to enforce

Steps:
1. A `review` task reads the file at `file_path` and the `standards` description, identifies
   violations, writes a violations report artifact, and returns `outcome` (`violations_found`
   or `approved`) and `violation_count`.
2. A `repeat_until` loop (max 3 iterations) that runs while violations are found:
   - A `fix` task reads the violations report from the previous review and the source file,
     applies fixes directly to the file in place, and returns `outcome` (`fixed`).
   - A `re_review` task re-reads the (now fixed) file and the `standards`, writes an updated
     violations report artifact, and returns `outcome` (`violations_found` or `approved`) and
     `violation_count`.
   - Loop exits when `re_review` returns `outcome` equal to `approved`.

Output artifacts:
- `violations_report` — the final violations report from the last review

Output results:
- `outcome` — final outcome from the last review (`approved` or `violations_found`)
- `violation_count` — number of remaining violations from the last review
