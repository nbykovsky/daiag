# New Poem

This example shows a single reusable task loaded from `.daiag/tasks`.

The workflow accepts two parameters:

- `topic`
- `line_count`

It writes the resulting poem to `examples/new_poem/poem.md`.

Run it from the repository root:

```sh
go run ./cmd/daiag run --workflow examples/new_poem/workflows/poem.star --param topic=starlight --param line_count=6
```

Prerequisites:

- `codex` CLI must be installed and authenticated

Expected output:

- `examples/new_poem/poem.md`
