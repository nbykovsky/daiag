# New Poem

This example shows a single inline task with a sibling prompt template.

The workflow accepts two parameters:

- `topic`
- `line_count`

It writes the resulting poem to `examples/new_poem/poem.md`.

Run it from the repository root:

```sh
go run ./cmd/daiag run --workflow new_poem --workflows-lib examples/new_poem/workflows --workdir "$PWD" --param topic=starlight --param line_count=6
```

Prerequisites:

- `codex` CLI must be installed and authenticated

Expected output:

- `examples/new_poem/poem.md`
