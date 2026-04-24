package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"sf-cli/internal/api"
	"sf-cli/internal/model"
	"sf-cli/internal/output"
)

const envBearerToken = "SF_BEARER_TOKEN"

type rootConfig struct {
	BaseURL string
	Token   string
}

func Run(args []string, stdout io.Writer) int {
	envelope := execute(args)
	if err := output.WriteJSON(stdout, envelope); err != nil {
		fallback := fmt.Sprintf("{\n  \"version\": \"v1\",\n  \"mode\": \"read_only\",\n  \"command\": %q,\n  \"ok\": false,\n  \"result\": null,\n  \"error\": {\n    \"code\": \"output_error\",\n    \"message\": %q\n  }\n}\n", envelope.Command, err.Error())
		_, _ = io.WriteString(stdout, fallback)
		return 1
	}

	if envelope.OK {
		return 0
	}
	return 1
}

func execute(args []string) model.Envelope {
	config, remaining, err := parseRootFlags(args)
	if err != nil {
		return errorEnvelope("", nil, "invalid_arguments", err.Error())
	}

	if len(remaining) == 0 {
		return errorEnvelope("", nil, "invalid_arguments", "missing command")
	}

	client, err := api.NewClient(api.Options{
		BaseURL: config.BaseURL,
		Token:   resolveToken(config.Token, os.Getenv(envBearerToken)),
	})
	if err != nil {
		return errorEnvelope("", nil, "invalid_configuration", err.Error())
	}

	switch remaining[0] {
	case "tickets":
		return handleTickets(context.Background(), client, remaining[1:])
	case "project":
		return handleProject(context.Background(), client, remaining[1:])
	case "tracker":
		return handleTracker(remaining[1:])
	default:
		return errorEnvelope(remaining[0], nil, "invalid_command", fmt.Sprintf("unknown command %q", remaining[0]))
	}
}

func handleProject(ctx context.Context, client *api.Client, args []string) model.Envelope {
	if len(args) == 0 {
		return errorEnvelope("project", proposal("project", "dispatch_project_command", nil, nil), "invalid_arguments", "missing project subcommand")
	}

	switch args[0] {
	case "tools":
		return runProjectTools(ctx, client, args[1:])
	default:
		command := "project." + args[0]
		return errorEnvelope(command, proposal(command, actionForProject(args[0]), nil, nil), "not_implemented", fmt.Sprintf("command %q is not implemented yet", command))
	}
}

func parseRootFlags(args []string) (rootConfig, []string, error) {
	fs := flag.NewFlagSet("sf", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := rootConfig{}
	fs.StringVar(&config.BaseURL, "base-url", api.DefaultBaseURL, "Base URL for the SourceForge REST API")
	fs.StringVar(&config.Token, "token", "", "Bearer token for authenticated requests")

	if err := fs.Parse(args); err != nil {
		return rootConfig{}, nil, normalizeFlagError(err)
	}

	return config, fs.Args(), nil
}

func handleTickets(ctx context.Context, client *api.Client, args []string) model.Envelope {
	if len(args) == 0 {
		return errorEnvelope("tickets", proposal("tickets", "dispatch_tickets_command", nil, nil), "invalid_arguments", "missing tickets subcommand")
	}

	switch args[0] {
	case "list":
		return runTicketsList(ctx, client, args[1:])
	case "search":
		return runTicketsSearch(ctx, client, args[1:])
	case "get":
		return runTicketsGet(ctx, client, args[1:])
	case "comments":
		return runTicketsComments(ctx, client, args[1:])
	default:
		command := "tickets." + args[0]
		return errorEnvelope(command, proposal(command, actionForTickets(args[0]), nil, nil), "not_implemented", fmt.Sprintf("command %q is not implemented yet", command))
	}
}

func handleTracker(args []string) model.Envelope {
	if len(args) == 0 {
		return errorEnvelope("tracker", proposal("tracker", "dispatch_tracker_command", nil, nil), "invalid_arguments", "missing tracker subcommand")
	}

	command := "tracker." + args[0]
	return errorEnvelope(command, proposal(command, actionForTracker(args[0]), nil, nil), "not_implemented", fmt.Sprintf("command %q is not implemented yet", command))
}

func proposal(command string, action string, target map[string]any, inputs map[string]any) *model.Proposal {
	return &model.Proposal{
		Action:  action,
		Target:  target,
		Inputs:  inputs,
		Effects: []model.Effect{},
	}
}

func errorEnvelope(command string, proposal *model.Proposal, code string, message string) model.Envelope {
	return model.Envelope{
		Version:  "v1",
		Mode:     "read_only",
		Command:  command,
		OK:       false,
		Proposal: proposal,
		Result:   nil,
		Error: &model.Error{
			Code:    code,
			Message: message,
		},
	}
}

func successEnvelope(command string, proposal *model.Proposal, result any) model.Envelope {
	return model.Envelope{
		Version:  "v1",
		Mode:     "read_only",
		Command:  command,
		OK:       true,
		Proposal: proposal,
		Result:   result,
		Error:    nil,
	}
}

func resolveToken(flagToken string, envToken string) string {
	if trimmed := strings.TrimSpace(flagToken); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(envToken)
}

func normalizeFlagError(err error) error {
	if errors.Is(err, flag.ErrHelp) {
		return errors.New("help output is not supported; use JSON command invocations")
	}
	return err
}

func actionForTickets(subcommand string) string {
	switch subcommand {
	case "list":
		return "list_tickets"
	case "search":
		return "search_tickets"
	case "get":
		return "get_ticket"
	case "comments":
		return "get_ticket_comments"
	default:
		return "dispatch_tickets_command"
	}
}

func actionForTracker(subcommand string) string {
	if subcommand == "schema" {
		return "get_tracker_schema"
	}
	return "dispatch_tracker_command"
}

func actionForProject(subcommand string) string {
	if subcommand == "tools" {
		return "list_project_tools"
	}
	return "dispatch_project_command"
}
