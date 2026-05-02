package cli

import "strings"

func helpText(args []string) (string, bool) {
	path, explicitHelp := helpPath(args)
	if !explicitHelp {
		return "", false
	}

	switch len(path) {
	case 0:
		return rootUsage(), true
	case 1:
		switch path[0] {
		case "tickets":
			return ticketsUsage(), true
		case "actions":
			return actionsUsage(), true
		case "project":
			return projectUsage(), true
		case "tracker":
			return trackerUsage(), true
		}
	case 2:
		switch path[0] + "." + path[1] {
		case "tickets.list":
			return ticketsListUsage("tickets list"), true
		case "tickets.search":
			return ticketsSearchUsage(), true
		case "tickets.activity":
			return ticketsActivityUsage(), true
		case "tickets.get":
			return ticketsGetUsage("tickets get"), true
		case "tickets.comments":
			return ticketsGetUsage("tickets comments"), true
		case "actions.validate":
			return actionsValidateUsage(), true
		case "actions.apply":
			return actionsApplyUsage(), true
		case "project.tools":
			return projectToolsUsage(), true
		case "tracker.schema":
			return trackerSchemaUsage(), true
		}
	}

	return rootUsage(), true
}

func helpPath(args []string) ([]string, bool) {
	path := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "help" {
			return append([]string{}, argsAfterHelp(args[i+1:])...), true
		}
		if isHelpFlag(arg) {
			return path, true
		}
		if len(path) == 0 && isRootFlag(arg) {
			if consumesNextValue(arg) && i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		path = append(path, arg)
		if len(path) == 2 {
			continue
		}
	}
	return nil, false
}

func argsAfterHelp(args []string) []string {
	path := make([]string, 0, 2)
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		path = append(path, arg)
		if len(path) == 2 {
			break
		}
	}
	return path
}

func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func isRootFlag(arg string) bool {
	return arg == "--base-url" || strings.HasPrefix(arg, "--base-url=") || arg == "--token" || strings.HasPrefix(arg, "--token=")
}

func consumesNextValue(arg string) bool {
	return arg == "--base-url" || arg == "--token"
}

func rootUsage() string {
	return strings.TrimSpace(`Usage:
  sf [--base-url URL] [--token TOKEN] <command> [args]

Purpose:
	  sf is a SourceForge CLI for discovering project and tracker metadata, reading
	  tracker tickets through stable JSON envelopes, and validating or safely
	  staging write intents before proposing external mutations.

Commands:
  tickets      List, search, inspect, and comment-read tracker tickets
	  actions      Validate and safely stage write intents
  project      Inspect project metadata
  tracker      Inspect tracker schema metadata
  help         Show help for a command

Global options:
  --base-url URL   Base URL for the SourceForge REST API
  --token TOKEN    Bearer token for authenticated requests

Environment:
  SF_BEARER_TOKEN  Bearer token used when --token is not provided

Output contract:
  Normal command execution returns a JSON envelope with these top-level fields:
  version, mode, command, ok, warnings, proposal, result, error

  Read envelopes in this order:
  1. Check ok
  2. If ok is false, inspect error
  3. If ok is true, consume result
  4. Treat proposal and warnings as supplemental metadata

Agent guidance:
	  Start by choosing the smallest command that answers the question:
	  - Use 'project tools' to discover tracker mount points for a project.
	  - Use 'tickets list' for pagination over a tracker without a search query.
	  - Use 'tickets search' when you need a SourceForge query such as
	    'status:open' or label filters.
	  - Use 'tickets activity' to review the most recently updated tickets. It
	    defaults to open tickets and accepts '--all' to include closed tickets.
	  - Use 'tickets get' for one ticket record and 'tickets comments' for the
	    discussion thread.
	  - Use 'tracker schema' before generating filters, field mappings, or
	    write-intent payloads.
	  - Use 'actions validate' before proposing or applying ticket create, label,
	    or comment writes.
	
  - Prefer explicit --project and --tracker flags instead of assuming defaults.
  - Use 'sf help <command>' before a new workflow to inspect required flags.
	  - Root help is an index; subcommand help is the source of truth for exact
	    flags, supported '--fields' values, and result shape.
  - Canonical ticket payloads use SourceForge-native field names; compact
	    --fields projections may use aliases such as 'id' and 'title'.
	  - Cursors are opaque numeric tokens. Reuse the returned value exactly as
	    provided in 'result.pagination.next'.
	  - Current releases remain mostly read-oriented. 'actions apply' now executes
	    confirmed 'ticket_create', 'ticket_comment', and 'ticket_labels' actions.

Command map:
  sf project tools
    Discover available project tools and tracker mount points.

  sf tickets list
    Enumerate tickets in a tracker.

  sf tickets search
    Run a SourceForge ticket query against one tracker.

  sf tickets activity
    List most recently updated tickets, open-only by default.

  sf tickets get
    Fetch one ticket with the canonical detail shape.

  sf tickets comments
    Fetch the ticket discussion thread and comments.

  sf tracker schema
    Fetch best-effort schema sections: project, tracker, options, milestones,
    saved_bins, and fields.

	  sf actions validate
	    Validate an actions JSON file for supported dry-run write intents.

	  sf actions apply
	    Run apply safety checks and require explicit confirmation before any
	    future write execution.

Common workflows:
  Discover project tools:
    sf project tools --project fuse-emulator

  List tickets in a tracker:
    sf tickets list --project fuse-emulator --tracker bugs

  Search open tickets:
    sf tickets search --project fuse-emulator --tracker bugs --query 'status:open'

  Review recently active tickets:
    sf tickets activity --project fuse-emulator --tracker bugs

  Inspect one ticket and then read its comments:
    sf tickets get --project fuse-emulator --tracker bugs --ticket 42
    sf tickets comments --project fuse-emulator --tracker bugs --ticket 42

  Inspect tracker schema before generating automation:
    sf tracker schema --project fuse-emulator --tracker bugs

  Validate write intents from an actions file:
    sf actions validate actions.json

Current write-intent support:
  - 'actions validate' accepts a JSON file containing an 'actions' array.
	  - Supported action types today are 'ticket_create', 'ticket_labels', and
	    'ticket_comment'.
	  - 'actions apply' reuses the same dry-run and confirmation safety plumbing,
	    requires bearer auth when --confirm is used, executes confirmed
	    'ticket_create', 'ticket_comment', and 'ticket_labels' actions, and still
	    rejects unsupported write action types.
  - 'ticket_create' validates new ticket drafts with required summary text,
    optional description, optional status, optional assignee, optional private
    flag, optional custom fields, and optional labels that do not contain
    commas. When omitted, status defaults to 'open'.
  - 'ticket_labels' validates replacement-style label updates with one or more
    non-empty labels that do not contain commas.
  - 'ticket_comment' validates new top-level ticket discussion posts only. It
    requires a non-empty body, an existing ticket, discussion enabled on that
    ticket, and a resolvable discussion thread id.
  - Validation reports per-action ok state, structured issues, normalized action
    data, and canonical identifiers when resolution succeeds.

Examples:
  sf help tickets
  sf help tickets search
  sf actions validate actions.json
  sf tickets activity --project fuse-emulator --tracker bugs --all
  sf tracker schema --project fuse-emulator --tracker bugs
`) + "\n"
}

func actionsUsage() string {
	return strings.TrimSpace(`Usage:
  sf actions <subcommand> [args]

Subcommands:
  validate    Validate write intents from an actions file
  apply       Run apply safety checks for an actions file

Notes:
  'actions validate' is a dry-run interface for automation. 'actions apply'
  layers confirmation-oriented safety checks on top of the same file validation
  path. Today the supported action types are 'ticket_create', 'ticket_labels',
  and 'ticket_comment' for validation and confirmed apply execution. See
  'sf help actions validate' for the exact input shape and 'sf help actions
  apply' for the apply safety model.

Examples:
  sf actions validate actions.json
  sf actions apply actions.json
  sf actions apply --confirm actions.json
`) + "\n"
}

func actionsValidateUsage() string {
	return strings.TrimSpace(`Usage:
  sf actions validate ACTIONS_FILE

Arguments:
  ACTIONS_FILE  JSON file containing an 'actions' array

Supported action types today:
  ticket_create   Validate a new ticket draft
  ticket_labels   Replace the ticket label set with a validated labels array
  ticket_comment  Validate a new top-level ticket discussion post

Expected input shape:
  {
    "actions": [
      {
        "type": "ticket_create",
        "project": "fuse-emulator",
        "tracker": "bugs",
        "summary": "Add deterministic export",
        "description": "Normalize timestamps before writing output",
        "status": "open",
        "private": false,
        "custom_fields": {"_priority": "5"},
        "labels": ["triaged", "needs-review"]
      },
      {
        "type": "ticket_labels",
        "project": "fuse-emulator",
        "tracker": "bugs",
        "ticket": 42,
        "labels": ["triaged", "needs-review"]
      },
      {
        "type": "ticket_comment",
        "project": "fuse-emulator",
        "tracker": "bugs",
        "ticket": 42,
        "body": "comment text"
      }
    ]
  }

Current ticket_create scope:
  - validates SourceForge-compatible create inputs for required 'summary',
    optional 'description', optional 'status', optional 'private', optional
    'custom_fields', and optional 'labels'
  - requires a non-empty 'summary'
  - defaults 'status' to 'open' when omitted
  - rejects 'assigned_to' and 'discussion_disabled'; they are not modeled for
    ticket_create
  - rejects labels containing commas because the SourceForge write API uses a
    comma-separated 'ticket_form.labels' field

Current ticket_labels scope:
  - validates replacement-style label updates only
  - requires one or more non-empty labels
  - rejects labels containing commas because the SourceForge write API uses a
    comma-separated 'ticket_form.labels' field

Current ticket_comment scope:
  - validates new top-level discussion posts only; reply posts are not modeled
    yet
  - requires a non-empty 'body'
  - requires the target ticket to exist, allow discussion, and expose a
    discussion thread id

Validation output:
  result.ok                 overall file validity
  result.validated_actions  per-action validation records

Per-action result fields:
  index                  original action index from the input file
  type                   action type from the input
  target                 requested target identifiers
  action                 normalized supported action payload when available
  canonical_identifiers  resolved canonical target identifiers when available
  ok                     per-action validity
  issues                 structured validation problems when ok is false
`) + "\n"
}

func actionsApplyUsage() string {
	return strings.TrimSpace(`Usage:
  sf actions apply [--confirm] ACTIONS_FILE

Arguments:
  ACTIONS_FILE  JSON file containing an 'actions' array

Options:
  --confirm     Allow apply to proceed past dry-run validation checks

Safety model:
  Without '--confirm', the command validates and previews only. This default
  mode performs the same action-file checks as 'sf actions validate' and stops
  before any execution path.

  With '--confirm', the command may continue into write execution once specific
  action types are enabled. Confirmation does not bypass validation. Invalid
  actions still fail before execution begins, and bearer authentication is
  required via '--token' or 'SF_BEARER_TOKEN'.

Current execution scope:
  Confirmed apply currently executes 'ticket_create', 'ticket_comment', and
  'ticket_labels' actions. Mixed files containing unsupported types are still
  rejected before any write request is sent.

Result shape:
  result.ok                 overall apply-stage success
  result.confirmed          whether '--confirm' was provided
  result.executed           whether any write steps were executed
  result.validated_actions  per-action validation records reused from validate
  result.applied_actions    per-action execution records when confirmation was requested
`) + "\n"
}

func ticketsUsage() string {
	return "Usage:\n  sf tickets <subcommand> [args]\n\nSubcommands:\n  list        List tickets in a tracker\n  search      Search tracker tickets with a query\n  activity    Show most recently active open tickets in a tracker\n  get         Fetch a single ticket\n  comments    Fetch comments for a ticket\n\nSelection guide:\n  list        Browse tracker pages without a query\n  search      Use when you need a SourceForge query string\n  activity    Review recently updated tickets; add --all for closed tickets too\n  get         Fetch one ticket record\n  comments    Fetch a ticket discussion thread\n\nExamples:\n  sf tickets list --project fuse-emulator --tracker bugs\n  sf tickets search --project fuse-emulator --tracker bugs --query 'status:open'\n  sf tickets activity --project fuse-emulator --tracker bugs\n  sf tickets activity --project fuse-emulator --tracker bugs --all\n  sf tickets get --project fuse-emulator --tracker bugs --ticket 42\n  sf tickets comments --project fuse-emulator --tracker bugs --ticket 42\n"
}

func ticketsListUsage(command string) string {
	return "Usage:\n  sf " + command + " --project PROJECT --tracker TRACKER [--cursor TOKEN] [--limit N] [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --cursor TOKEN     Opaque cursor returned by a previous response\n  --limit N          Page size (default 25)\n  --fields LIST      Comma-separated compact fields for each returned item\n\nSupported --fields values:\n  id,title,status,reported_by,assigned_to,labels,created_at,updated_at\n\nResult shape:\n  Default output returns canonical ticket objects under result.tickets. When\n  --fields is provided, each item only contains the requested compact fields.\n"
}

func ticketsSearchUsage() string {
	return "Usage:\n  sf tickets search --project PROJECT --tracker TRACKER --query QUERY [--cursor TOKEN] [--limit N] [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --query QUERY      Ticket search query\n  --cursor TOKEN     Opaque cursor returned by a previous response\n  --limit N          Page size (default 25)\n  --fields LIST      Comma-separated compact fields for each returned ticket\n\nSupported --fields values:\n  id,title,status,reported_by,assigned_to,labels,created_at,updated_at\n\nResult shape:\n  The result includes tickets, count, limit, pagination, and may also include\n  sort and filter_choices when SourceForge returns them.\n"
}

func ticketsActivityUsage() string {
	return "Usage:\n  sf tickets activity --project PROJECT --tracker TRACKER [--cursor TOKEN] [--limit N] [--fields LIST] [--all]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --cursor TOKEN     Opaque cursor returned by a previous response\n  --limit N          Page size (default 25)\n  --fields LIST      Comma-separated compact fields for each returned item\n  --all              Include closed issues; default is open issues only\n\nSupported --fields values:\n  id,title,status,updated_at,last_comment_at,last_comment_author\n\nResult shape:\n  Returns a tickets list sorted by most recent update first. The default output\n  contains activity-specific ticket objects under result.tickets.\n"
}

func ticketsGetUsage(command string) string {
	fields := "id,title,description,status,reported_by,assigned_to,labels,private,discussion_disabled,custom_fields,attachments,related_artifacts,created_at,updated_at"
	if command == "tickets comments" {
		fields = "id,author,body,created_at,edited_at,subject,type,is_meta,attachments"
	}
	return "Usage:\n  sf " + command + " --project PROJECT --tracker TRACKER --ticket NUMBER [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --ticket NUMBER    Ticket number\n  --fields LIST      Comma-separated compact fields for the returned item\n\nSupported --fields values:\n  " + fields + "\n"
}

func projectUsage() string {
	return "Usage:\n  sf project <subcommand> [args]\n\nSubcommands:\n  tools       List project tools and mount points\n\nExample:\n  sf project tools --project fuse-emulator\n"
}

func projectToolsUsage() string {
	return "Usage:\n  sf project tools --project PROJECT [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --fields LIST      Comma-separated compact fields for each returned tool\n\nSupported --fields values:\n  name,mount_point,mount_label,url,api_url,clone_url_https_anon,clone_url_ro\n\nResult shape:\n  Default output returns the full project payload, including its tools array.\n  With --fields, result.tools contains only the requested tool fields.\n"
}

func trackerUsage() string {
	return "Usage:\n  sf tracker <subcommand> [args]\n\nSubcommands:\n  schema      Show best-effort tracker schema metadata\n\nExample:\n  sf tracker schema --project fuse-emulator --tracker bugs\n"
}

func trackerSchemaUsage() string {
	return "Usage:\n  sf tracker schema --project PROJECT --tracker TRACKER [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --fields LIST      Comma-separated top-level schema sections to return\n\nSupported --fields values:\n  project,tracker,options,milestones,saved_bins,fields\n\nNotes:\n  This is best-effort metadata assembled from SourceForge responses. Warnings may\n  be returned alongside successful results when some schema sections are partial\n  or unavailable.\n"
}
