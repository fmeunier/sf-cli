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
	return "Usage:\n  sf [--base-url URL] [--token TOKEN] <command> [args]\n\nCommands:\n  tickets      List, search, inspect, and comment-read tracker tickets\n  actions      Dry-run validation for write intents\n  project      Inspect project metadata\n  tracker      Inspect tracker schema metadata\n  help         Show help for a command\n\nGlobal options:\n  --base-url URL   Base URL for the SourceForge REST API\n  --token TOKEN    Bearer token for authenticated requests\n\nEnvironment:\n  SF_BEARER_TOKEN  Bearer token used when --token is not provided\n\nExamples:\n  sf help tickets\n  sf actions validate actions.json\n  sf tickets list --project fuse-emulator --tracker bugs\n  sf tracker schema --project fuse-emulator --tracker bugs\n"
}

func actionsUsage() string {
	return "Usage:\n  sf actions <subcommand> [args]\n\nSubcommands:\n  validate    Validate write intents from an actions file\n\nExample:\n  sf actions validate actions.json\n"
}

func actionsValidateUsage() string {
	return "Usage:\n  sf actions validate ACTIONS_FILE\n\nArguments:\n  ACTIONS_FILE  JSON file containing an `actions` array\n"
}

func ticketsUsage() string {
	return "Usage:\n  sf tickets <subcommand> [args]\n\nSubcommands:\n  list        List tickets in a tracker\n  search      Search tracker tickets with a query\n  activity    Show most recently active tickets in a tracker\n  get         Fetch a single ticket\n  comments    Fetch comments for a ticket\n\nExamples:\n  sf tickets list --project fuse-emulator --tracker bugs\n  sf tickets search --project fuse-emulator --tracker bugs --query 'status:open'\n  sf tickets activity --project fuse-emulator --tracker bugs\n  sf tickets get --project fuse-emulator --tracker bugs --ticket 42\n  sf tickets comments --project fuse-emulator --tracker bugs --ticket 42\n"
}

func ticketsListUsage(command string) string {
	return "Usage:\n  sf " + command + " --project PROJECT --tracker TRACKER [--page N] [--limit N]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --page N           Result page to fetch (default 0)\n  --limit N          Page size (default 25)\n"
}

func ticketsSearchUsage() string {
	return "Usage:\n  sf tickets search --project PROJECT --tracker TRACKER --query QUERY [--page N] [--limit N]\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --query QUERY      Ticket search query\n  --page N           Result page to fetch (default 0)\n  --limit N          Page size (default 25)\n"
}

func ticketsGetUsage(command string) string {
	return "Usage:\n  sf " + command + " --project PROJECT --tracker TRACKER --ticket NUMBER\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n  --ticket NUMBER    Ticket number\n"
}

func projectUsage() string {
	return "Usage:\n  sf project <subcommand> [args]\n\nSubcommands:\n  tools       List project tools and mount points\n\nExample:\n  sf project tools --project fuse-emulator\n"
}

func projectToolsUsage() string {
	return "Usage:\n  sf project tools --project PROJECT\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n"
}

func trackerUsage() string {
	return "Usage:\n  sf tracker <subcommand> [args]\n\nSubcommands:\n  schema      Show best-effort tracker schema metadata\n\nExample:\n  sf tracker schema --project fuse-emulator --tracker bugs\n"
}

func trackerSchemaUsage() string {
	return "Usage:\n  sf tracker schema --project PROJECT --tracker TRACKER\n\nArguments:\n  --project PROJECT  SourceForge project shortname\n  --tracker TRACKER  Tracker mount point\n"
}
