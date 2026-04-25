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
  sf is a SourceForge-focused CLI for machine-readable read workflows plus dry-run
  validation of write intents. Prefer this CLI when an agent needs stable JSON
  envelopes instead of scraping HTML or ad-hoc text.

Commands:
  tickets      List, search, inspect, and comment-read tracker tickets
  actions      Dry-run validation for write intents
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
  - Prefer explicit --project and --tracker flags instead of assuming defaults.
  - Use 'sf help <command>' before a new workflow to inspect required flags.
  - Use 'project tools' to discover valid tracker mount points for a project.
  - Use 'tracker schema' to inspect tracker fields before generating queries or
    write intents.
  - Use 'actions validate' before proposing or applying ticket-comment writes.
  - Cursors are opaque. Reuse the returned token exactly as provided.
  - This CLI is read-only except for dry-run validation; it does not post
    comments or mutate SourceForge state.

Common workflows:
  Discover project tools:
    sf project tools --project fuse-emulator

  List tickets in a tracker:
    sf tickets list --project fuse-emulator --tracker bugs

  Search open tickets:
    sf tickets search --project fuse-emulator --tracker bugs --query 'status:open'

  Inspect one ticket and then read its comments:
    sf tickets get --project fuse-emulator --tracker bugs --ticket 42
    sf tickets comments --project fuse-emulator --tracker bugs --ticket 42

  Inspect tracker schema before generating automation:
    sf tracker schema --project fuse-emulator --tracker bugs

  Validate write intents from an actions file:
    sf actions validate actions.json

Current write-intent support:
  - 'actions validate' accepts a JSON file containing an 'actions' array.
  - The first supported action type is 'ticket_comment'.
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
	return "Usage:\n  sf actions <subcommand> [args]\n\nSubcommands:\n  validate    Validate write intents from an actions file\n\nNotes:\n  `actions validate` is a dry-run interface for automation. It does not post or\n  modify SourceForge data. Today the supported action type is `ticket_comment`.\n\nExample:\n  sf actions validate actions.json\n"
}

func actionsValidateUsage() string {
	return "Usage:\n  sf actions validate ACTIONS_FILE\n\nArguments:\n  ACTIONS_FILE  JSON file containing an `actions` array\n\nExpected input shape:\n  {\n    \"actions\": [\n      {\n        \"type\": \"ticket_comment\",\n        \"project\": \"fuse-emulator\",\n        \"tracker\": \"bugs\",\n        \"ticket\": 42,\n        \"body\": \"comment text\"\n      }\n    ]\n  }\n\nValidation output:\n  result.ok                 Overall validation success across all actions\n  result.validated_actions  Per-action validation results\n\nPer-action result fields:\n  index                  Input position in the actions array\n  type                   Original action type\n  target                 Original target fields\n  action                 Normalized supported action data\n  canonical_identifiers  Resolved canonical identifiers when available\n  ok                     Action-specific validation success\n  issues                 Structured warnings and errors\n"
}

func ticketsUsage() string {
	return "Usage:\n  sf tickets <subcommand> [args]\n\nSubcommands:\n  list        List tickets in a tracker\n  search      Search tracker tickets with a query\n  activity    Show most recently active open tickets in a tracker\n  get         Fetch a single ticket\n  comments    Fetch comments for a ticket\n\nExamples:\n  sf tickets list --project fuse-emulator --tracker bugs\n  sf tickets search --project fuse-emulator --tracker bugs --query 'status:open'\n  sf tickets activity --project fuse-emulator --tracker bugs\n  sf tickets activity --project fuse-emulator --tracker bugs --all\n  sf tickets get --project fuse-emulator --tracker bugs --ticket 42\n  sf tickets comments --project fuse-emulator --tracker bugs --ticket 42\n"
}

func ticketsListUsage(command string) string {
	return "Usage:\n  sf " + command + " --project PROJECT --tracker TRACKER [--cursor TOKEN] [--limit N] [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --cursor TOKEN     Opaque cursor returned by a previous response\n  --limit N          Page size (default 25)\n  --fields LIST      Comma-separated compact fields for each returned item\n"
}

func ticketsSearchUsage() string {
	return "Usage:\n  sf tickets search --project PROJECT --tracker TRACKER --query QUERY [--cursor TOKEN] [--limit N] [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --query QUERY      Ticket search query\n  --cursor TOKEN     Opaque cursor returned by a previous response\n  --limit N          Page size (default 25)\n  --fields LIST      Comma-separated compact fields for each returned ticket\n"
}

func ticketsActivityUsage() string {
	return "Usage:\n  sf tickets activity --project PROJECT --tracker TRACKER [--cursor TOKEN] [--limit N] [--fields LIST] [--all]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --cursor TOKEN     Opaque cursor returned by a previous response\n  --limit N          Page size (default 25)\n  --fields LIST      Comma-separated compact fields for each returned item\n  --all              Include closed issues; default is open issues only\n"
}

func ticketsGetUsage(command string) string {
	return "Usage:\n  sf " + command + " --project PROJECT --tracker TRACKER --ticket NUMBER [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --ticket NUMBER    Ticket number\n  --fields LIST      Comma-separated compact fields for the returned item\n"
}

func projectUsage() string {
	return "Usage:\n  sf project <subcommand> [args]\n\nSubcommands:\n  tools       List project tools and mount points\n\nExample:\n  sf project tools --project fuse-emulator\n"
}

func projectToolsUsage() string {
	return "Usage:\n  sf project tools --project PROJECT [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --fields LIST      Comma-separated compact fields for each returned tool\n"
}

func trackerUsage() string {
	return "Usage:\n  sf tracker <subcommand> [args]\n\nSubcommands:\n  schema      Show best-effort tracker schema metadata\n\nExample:\n  sf tracker schema --project fuse-emulator --tracker bugs\n"
}

func trackerSchemaUsage() string {
	return "Usage:\n  sf tracker schema --project PROJECT --tracker TRACKER [--fields LIST]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --fields LIST      Comma-separated top-level schema sections to return\n"
}
