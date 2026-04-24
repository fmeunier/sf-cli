# sf-cli

`sf-cli` is a small Go command-line client for the SourceForge Allura REST API.

The current MVP focuses on read-only JSON workflows for:
- listing tracker tickets
- searching tracker tickets
- fetching a single ticket
- fetching ticket comments
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

## Scope And Limits

Current scope:
- SourceForge Allura REST read-only workflows
- JSON envelope output for normal command execution
- best-effort tracker schema metadata when upstream data is partial or inconsistent

Not included in this MVP:
- docker runner support
- repository clone/fetch support
- write operations such as creating or editing tickets
