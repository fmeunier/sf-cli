# sf-cli

`sf-cli` is a small Go command-line client for the SourceForge Allura REST API.

The current MVP focuses on read-only JSON workflows for:
- listing tracker tickets
- searching tracker tickets
- fetching a single ticket
- fetching ticket comments
- validating write-intent action files without side effects
- inspecting project tools
- inspecting best-effort tracker schema metadata

## Build

Requirements:
- Go 1.26+

Build the CLI:

```bash
go build -o sf ./cmd/sf
```

Run verification:

```bash
go test ./...
go build ./...
```

## Authentication

Authenticated requests use a SourceForge bearer token.

You can pass it explicitly:

```bash
sf --token "$SF_BEARER_TOKEN" ...
```

Or set the environment variable used by the CLI:

```bash
export SF_BEARER_TOKEN="your-token"
```

The `--token` flag takes precedence over `SF_BEARER_TOKEN`.

You can also override the API base URL when testing against another Allura instance:

```bash
sf --base-url https://sourceforge.net/rest ...
```

## Help

The CLI supports standard help entry points:

```bash
sf --help
sf help tickets
sf tickets search --help
```

## Commands

List tickets:

```bash
sf tickets list --project fuse-emulator --tracker bugs
```

Search tickets:

```bash
sf tickets search --project fuse-emulator --tracker bugs --query 'status:open'
```

Get one ticket:

```bash
sf tickets get --project fuse-emulator --tracker bugs --ticket 42
```

Get ticket comments:

```bash
sf tickets comments --project fuse-emulator --tracker bugs --ticket 42
```

Validate a dry-run actions file:

```bash
sf actions validate actions.json
```

List project tools:

```bash
sf project tools --project fuse-emulator
```

Inspect tracker schema metadata:

```bash
sf tracker schema --project fuse-emulator --tracker bugs
```

## Output

Normal command execution returns JSON envelopes.

Successful calls return fields like:
- `version`
- `mode`
- `command`
- `ok`
- `warnings`
- `proposal`
- `result`
- `error`

Failures return the same envelope shape with `ok: false`, `result: null`, and an `error` object containing stable `code` and `message` fields.

Warnings are reported at the top-level `warnings` field so callers do not need to inspect command-specific payloads for partial-success metadata.

For ticket reads, overlapping ticket fields use the same names and shapes across `tickets list`, `tickets search`, and `tickets get`. Collection responses return those ticket objects in `result.tickets`, while detail responses return one ticket object in `result.ticket`.

`tickets comments` returns normalized comment data in `result.comments`, ordered by `created_at` ascending and then `id` ascending when timestamps are equal or missing. Each comment uses the same shape: `id`, `author`, `body`, `created_at`, `edited_at`, `subject`, `is_meta`, and `attachments`. Minimal thread metadata remains in `result.thread`.

Paginated collection commands expose `result.pagination` with `page`, `limit`, `count`, `has_previous`, `has_next`, `previous_page`, and `next_page`. Unpaginated collection commands omit `result.pagination` entirely.

`tracker schema` keeps best-effort field values and now also exposes `fields[].validation` with structured validation metadata where the upstream tracker data permits it. Today that includes inferred field `type`, normalized `allowed_values`, and best-effort `default` values such as the default milestone when SourceForge exposes one.

`actions validate` reads a JSON file with an `actions` array and returns machine-readable validation results in `result.ok` and `result.validated_actions`. The first supported dry-run intent is `ticket_comment`, which checks tracker existence, ticket existence, and comment body length without applying any changes.

## Scope And Limits

Current scope:
- SourceForge Allura REST read-only workflows
- JSON envelope output for normal command execution
- best-effort tracker schema metadata when upstream data is partial or inconsistent

Not included in this MVP:
- docker runner support
- repository clone/fetch support
- write operations such as creating or editing tickets
