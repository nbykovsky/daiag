# Indicators PRD

## Goal

Add an `indicators` feature to the CLI so users can request derived market indicators
for a symbol over a chosen time range.

## Requirements

- Add a user-facing command for indicators.
- Support at least one symbol input.
- Validate required inputs clearly.
- Return machine-readable output.
- Document the command after implementation.

## Notes

- Exact indicator set is intentionally left open for the spec writer to refine.
- The implementation should follow existing repository patterns instead of inventing a new CLI shape.
