# sf-cli

`sf-cli` is a small Go command-line client for the SourceForge Allura REST API.

The current MVP focuses on read-only JSON workflows for:
- listing tracker tickets
- searching tracker tickets
- listing recently active tracker tickets
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

Show recently active tickets:

```bash
sf tickets activity --project fuse-emulator --tracker bugs
```

Get one ticket:

```bash
sf tickets get --project fuse-emulator --tracker bugs --ticket 42
```

Get a compact ticket payload:

```bash
sf tickets get --project fuse-emulator --tracker bugs --ticket 42 --fields id,title,status,updated_at
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

For ticket reads, collection commands return ticket objects in `result.tickets`, while the detail command returns one ticket object in `result.ticket`.

The canonical ticket contract is:
- `tickets list` and `tickets search` always return `ticket_num`, `summary`, and `status`.
- `tickets list` and `tickets search` may also return `reported_by`, `assigned_to`, `labels`, `created_date`, and `mod_date` when SourceForge provides meaningful values.
- `tickets get` always returns `ticket_num`, `summary`, `description`, `status`, `private`, `discussion_disabled`, and `discussion_thread`.
- `tickets get` may also return `reported_by`, `assigned_to`, `labels`, `created_date`, `mod_date`, `discussion_thread_url`, `custom_fields`, `attachments`, and `related_artifacts` when SourceForge provides meaningful values.
- Ticket payloads do not emit JSON `null` today. Optional fields are omitted when they are empty, and detail-only fields must not appear in `tickets list` or `tickets search`.

Compatible schema changes should be additive, start as optional fields, and update the documentation plus the conformance tests before widening command coverage.

`tickets activity` returns tickets ordered by most recent activity. Each activity entry includes `activity_type` plus `updated_at`, `last_comment_at`, and `last_comment_author` when comment metadata is available.

The `tickets activity` contract for `activity_type` is:
- `comment`: the latest known activity is a normalized ticket comment, based on a comment timestamp that is newer than or equal to the ticket `mod_date`.
- `ticket`: the latest known activity is the ticket record itself, either because there is no comment activity or because the ticket `mod_date` is newer than the latest normalized comment timestamp.
- `unknown`: a fallback used when SourceForge does not provide enough timestamp detail to determine whether the latest activity came from the ticket record or a comment.

This mapping is best-effort because SourceForge does not expose a first-class activity event type in ticket activity responses. `sf-cli` derives `activity_type` from the ticket `mod_date` plus the normalized latest comment timestamp when available, and falls back to `unknown` when those provider-specific signals are too sparse or ambiguous.

`tickets comments` returns normalized comment data in `result.comments`, ordered by `created_at` ascending and then `id` ascending when timestamps are equal or missing. Each comment uses the same shape: `id`, `author`, `body`, `created_at`, `edited_at`, `subject`, `type`, `is_meta`, and `attachments`. `type` is the normalized classification: `system` for SourceForge meta/system posts (`is_meta: true`), `user` for non-meta posts with recognizable user-authored content, and `unknown` as a fallback for provider-specific or ambiguous post forms that do not clearly map to either class. Minimal thread metadata remains in `result.thread`.

Most read/query commands also accept `--fields` to return a compact projection instead of the full repeated payload. For ticket-oriented commands, compact field names use the shorter aliases `id`, `title`, `created_at`, and `updated_at`.

Paginated collection commands expose cursor-based `result.pagination` with `limit`, `count`, `next_cursor`, and `has_more`. Request the next page with `--cursor` using the opaque token returned by a previous response. Unpaginated collection commands omit `result.pagination` entirely.

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
