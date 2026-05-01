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

`sf --help` is intentionally written to be useful to coding agents. It explains
the JSON envelope contract, common workflows, current limits, and how to
discover the right subcommands before automating against the CLI, including
`tickets activity` for reviewing recently active work.

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

Preview an apply run without executing writes:

```bash
sf actions apply actions.json
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

Every envelope includes these top-level fields:
- `version`
- `mode`
- `command`
- `ok`
- `warnings`
- `proposal`
- `result`
- `error`

The envelope field semantics are:
- `result`: the primary machine-readable outcome for the command. On success, this is the command-specific payload. On failure, it is always `null`.
- `proposal`: optional machine-readable intent metadata for the command, including the normalized action plus resolved target and inputs. When no proposal metadata is available, the field is omitted entirely.
- `warnings`: non-fatal caveats about an otherwise successful response. Warnings never replace `result` and do not change `ok`; callers should treat them as supplemental metadata.

Coexistence and precedence rules:
- `result` and `proposal` may appear together on successful responses. `proposal` explains intent; `result` remains the source of truth for the command outcome.
- `warnings` may appear alongside `result`, alongside `proposal`, or alongside both.
- `proposal` may also appear on failures when the CLI can still report the resolved command intent. In that case `ok` is `false`, `result` is `null`, and `error` is authoritative.
- Consumers should interpret fields in this order: first `ok`, then `error` when `ok` is `false`, otherwise `result`, with `proposal` and `warnings` treated as contextual metadata.

Examples:

Successful result with proposal:

```json
{
  "ok": true,
  "warnings": [],
  "proposal": {
    "action": "list_tickets",
    "target": {"project": "fuse-emulator", "tracker": "bugs"},
    "inputs": {"limit": 10},
    "effects": []
  },
  "result": {
    "tickets": [],
    "count": 0
  },
  "error": null
}
```

Successful response with warnings:

```json
{
  "ok": true,
  "warnings": ["comment body exceeds the recommended length"],
  "result": {
    "ok": true,
    "validated_actions": []
  },
  "error": null
}
```

Failure with proposal context:

```json
{
  "ok": false,
  "warnings": [],
  "proposal": {
    "action": "get_ticket",
    "target": {"project": "fuse-emulator", "tracker": "bugs", "ticket": 42},
    "effects": []
  },
  "result": null,
  "error": {
    "code": "not_found",
    "message": "ticket not found"
  }
}
```

Warnings are reported at the top-level `warnings` field so callers do not need to inspect command-specific payloads for partial-success metadata.

For ticket reads, collection commands return ticket objects in `result.tickets`, while the detail command returns one ticket object in `result.ticket`.

Naming decision:
- Canonical ticket payloads remain SourceForge-native. When a command returns the full ticket schema without `--fields`, names like `ticket_num`, `summary`, `created_date`, and `mod_date` are the stable contract.
- Shorter names like `id`, `title`, `created_at`, and `updated_at` are compact projection aliases for `--fields` responses only. They are convenience names, not a second canonical schema.
- Any future move to normalized names as the canonical ticket schema would be a breaking contract change and should ship as an explicit follow-up, not as a silent rename.

The canonical ticket contract is:
- `tickets list` and `tickets search` always return `ticket_num`, `summary`, and `status`.
- `tickets list` and `tickets search` may also return `reported_by`, `assigned_to`, `labels`, `created_date`, and `mod_date` when SourceForge provides meaningful values.
- `tickets get` always returns `ticket_num`, `summary`, `description`, `status`, `private`, `discussion_disabled`, and `discussion_thread`.
- `tickets get` may also return `reported_by`, `assigned_to`, `labels`, `created_date`, `mod_date`, `discussion_thread_url`, `custom_fields`, `attachments`, and `related_artifacts` when SourceForge provides meaningful values.
- `tickets get` keeps `discussion_thread` limited to thread metadata needed to fetch comments later. Embedded discussion posts are omitted; use `tickets comments` for comment bodies and normalized comment types.
- Ticket payloads do not emit JSON `null` today. Optional fields are omitted when they are empty, and detail-only fields must not appear in `tickets list` or `tickets search`.

The canonical identifier contract is:
- `ticket_num` is the canonical ticket identifier across `tickets list`, `tickets search`, `tickets get`, and `tickets activity`.
- When callers request compact ticket projections with `--fields`, the alias `id` is the same logical identifier as `ticket_num`; it is not a separate ID namespace.
- Likewise, `title`, `created_at`, and `updated_at` are aliases for canonical `summary`, `created_date`, and `mod_date` in compact projections only.
- `tickets comments` is correlated to a ticket by the requested ticket number plus the normalized thread object in `result.thread`. `result.thread.id` is the normalized discussion-thread identifier for that ticket's comments surface.
- Comment `id` values identify comments within the discussion thread. They are not ticket IDs and must not be joined to `ticket_num`.
- Provider-specific thread identifiers such as `discussion_thread._id` and `discussion_thread.discussion_id` are secondary identifiers for the discussion surface, not replacements for the canonical ticket identifier.

Cross-surface correlation rules:
- Join ticket records from `tickets list`, `tickets search`, `tickets get`, and `tickets activity` by `ticket_num`, or by compact `id` after normalizing that alias back to `ticket_num`.
- Use `tickets get` when a client needs provider thread metadata such as `discussion_thread._id` or `discussion_thread.discussion_id`.
- Use `tickets comments` when a client needs normalized thread and comment data. `result.thread.id` should match the provider thread `_id` exposed by `tickets get` when both are present for the same ticket.
- Do not use comment IDs, thread IDs, or provider `discussion_id` values as ticket keys. They identify discussion resources related to a ticket, not the ticket record itself.

Compatible schema changes should be additive, start as optional fields, and update the documentation plus the conformance tests before widening command coverage.

`tickets activity` is a compact updated-tickets view. By default it returns open and pending tickets only, ordered by SourceForge `mod_date` descending. Use `--all` to include closed tickets in the same order.

Each activity entry uses the same ticket identifier contract as the other ticket read surfaces and currently includes `ticket_num`, `summary`, `status`, and `updated_at` in the canonical output.

`tickets comments` returns normalized comment data in `result.comments`, ordered by `created_at` ascending and then `id` ascending when timestamps are equal or missing. Each comment uses the same shape: `id`, `author`, `body`, `created_at`, `edited_at`, `subject`, `type`, `is_meta`, and `attachments`. `type` is the normalized classification: `system` for SourceForge meta/system posts (`is_meta: true`) and `human` for all non-meta posts. Provider-specific or ambiguous non-meta comment forms collapse to `human` so the contract stays small and stable. Minimal thread metadata remains in `result.thread`.

Most read/query commands also accept `--fields` to return a compact projection instead of the full repeated payload. For ticket-oriented commands, compact field names use the shorter aliases `id`, `title`, `created_at`, and `updated_at`.

Paginated collection commands expose cursor-based `result.pagination` with `limit`, `count`, `next_cursor`, and `has_more`. Request the next page with `--cursor` using the opaque token returned by a previous response. Unpaginated collection commands omit `result.pagination` entirely.

Pagination ordering and continuity rules:
- `tickets list` preserves the provider's default tracker-list ordering. `sf-cli` does not currently add a stronger ordering guarantee on top of SourceForge for that command.
- `tickets search` preserves the provider's search ordering. When the upstream response includes a `sort` value, that echoed value is the authoritative ordering description for the returned page.
- `tickets activity` requests SourceForge search results ordered by `mod_date_dt desc`. Its page boundaries therefore follow SourceForge's updated-ticket ordering, not a client-side reshuffle.
- Cursor tokens are opaque CLI cursors backed by SourceForge page-based pagination today. They identify the next page in the current ordered result set; they are not snapshot IDs and should not be treated as durable resume tokens across arbitrary data churn.
- There is no snapshot isolation across pages. If tickets are created, updated, reopened, or closed while a client is paginating, later pages may shift and a client can observe duplicates, skips, or records moving between pages.
- Clients that need continuity across multiple pages should deduplicate by canonical `ticket_num`, tolerate missing or shifted records, and prefer re-running a fresh query when strict recency matters more than exhaustive traversal.

`tracker schema` keeps best-effort field values and now also exposes `fields[].validation` with structured validation metadata where the upstream tracker data permits it. Today that includes inferred field `type`, normalized `allowed_values`, and best-effort `default` values such as the default milestone when SourceForge exposes one.

`actions validate` reads a JSON file with an `actions` array and returns machine-readable validation results in `result.ok` and `result.validated_actions`. Each `validated_actions` entry preserves the existing validation summary fields and now also includes normalized supported action data in `action` plus resolved canonical identifiers in `canonical_identifiers` when available. Invalid actions still report per-action `issues` so automation can repair and retry selectively. Supported dry-run intents today are `ticket_create`, `ticket_labels`, and `ticket_comment`. The current `ticket_create` slice validates the SourceForge-documented tracker-create inputs for `summary`, optional `description`, and optional `labels`; it requires a non-empty summary and rejects labels containing commas because SourceForge's write API accepts labels as a comma-separated `ticket_form.labels` field. For now it does not model `status`, `assigned_to`, `private`, `discussion_disabled`, or `custom_fields`. The current `ticket_labels` slice validates replacement-style label updates only with the same comma restriction. `ticket_comment` currently validates the SourceForge-documented new-post flow for an existing ticket discussion thread: a non-empty body on an existing ticket with discussion enabled and a resolvable `discussion_thread._id`. Reply-post intents are not modeled yet, so `ticket_comment` is intentionally limited to new top-level discussion posts.

`actions apply` now provides the shared apply scaffold for later write rollouts. Without `--confirm`, it stays in `mode: "dry_run"` and returns the same validated action records plus apply-stage metadata so automation can preview a future write run safely. With `--confirm`, validation still runs first, bearer authentication is required via `--token` or `SF_BEARER_TOKEN`, and the current release still rejects all write action types because no mutating handlers are enabled yet.

## Scope And Limits

Current scope:
- SourceForge Allura REST read-only workflows
- apply dry-run and confirmation safety scaffolding for future write support
- JSON envelope output for normal command execution
- best-effort tracker schema metadata when upstream data is partial or inconsistent

Not included in this MVP:
- docker runner support
- repository clone/fetch support
- enabled write operations such as creating or editing tickets
