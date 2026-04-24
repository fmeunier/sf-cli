package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"sf-cli/internal/api"
	"sf-cli/internal/model"
)

type ticketsListConfig struct {
	Project string
	Tracker string
	Page    int
	Limit   int
}

type ticketsSearchConfig struct {
	Project string
	Tracker string
	Query   string
	Page    int
	Limit   int
}

func runTicketsList(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsListFlags(args)
	command := "tickets.list"
	prop := proposal(command, "list_tickets", map[string]any{"project": config.Project, "tracker": config.Tracker}, map[string]any{"page": config.Page, "limit": config.Limit})
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.ListTickets(ctx, api.ListTicketsParams{
		Project: config.Project,
		Tracker: config.Tracker,
		Page:    config.Page,
		Limit:   config.Limit,
	})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

func runTicketsSearch(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsSearchFlags(args)
	command := "tickets.search"
	prop := proposal(command, "search_tickets", map[string]any{"project": config.Project, "tracker": config.Tracker}, map[string]any{"query": config.Query, "page": config.Page, "limit": config.Limit})
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.SearchTickets(ctx, api.SearchTicketsParams{
		Project: config.Project,
		Tracker: config.Tracker,
		Query:   config.Query,
		Page:    config.Page,
		Limit:   config.Limit,
	})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

func parseTicketsListFlags(args []string) (ticketsListConfig, error) {
	if hasFlag(args, "query") {
		return ticketsListConfig{}, fmt.Errorf("--query is only supported by `tickets search`; use `sf tickets search --project <project> --tracker <tracker> --query <query>`")
	}

	fs := flag.NewFlagSet("tickets list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := ticketsListConfig{}
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.IntVar(&config.Page, "page", 0, "Result page to fetch")
	fs.IntVar(&config.Limit, "limit", 25, "Page size")

	if err := fs.Parse(args); err != nil {
		return ticketsListConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return ticketsListConfig{}, err
	}
	if err := validatePagination(config.Page, config.Limit); err != nil {
		return ticketsListConfig{}, err
	}

	return config, nil
}

func parseTicketsSearchFlags(args []string) (ticketsSearchConfig, error) {
	fs := flag.NewFlagSet("tickets search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := ticketsSearchConfig{}
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.StringVar(&config.Query, "query", "", "Ticket search query")
	fs.IntVar(&config.Page, "page", 0, "Result page to fetch")
	fs.IntVar(&config.Limit, "limit", 25, "Page size")

	if err := fs.Parse(args); err != nil {
		return ticketsSearchConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return ticketsSearchConfig{}, err
	}
	if strings.TrimSpace(config.Query) == "" {
		return ticketsSearchConfig{}, fmt.Errorf("missing required --query")
	}
	if err := validatePagination(config.Page, config.Limit); err != nil {
		return ticketsSearchConfig{}, err
	}

	config.Query = strings.TrimSpace(config.Query)
	return config, nil
}

func validateTrackerTarget(project string, tracker string) error {
	if strings.TrimSpace(project) == "" {
		return fmt.Errorf("missing required --project")
	}
	if strings.TrimSpace(tracker) == "" {
		return fmt.Errorf("missing required --tracker")
	}
	return nil
}

func validatePagination(page int, limit int) error {
	if page < 0 {
		return fmt.Errorf("--page must be >= 0")
	}
	if limit <= 0 {
		return fmt.Errorf("--limit must be > 0")
	}
	return nil
}

func hasFlag(args []string, name string) bool {
	short := "-" + name
	long := "--" + name
	for _, arg := range args {
		if arg == short || arg == long || strings.HasPrefix(arg, short+"=") || strings.HasPrefix(arg, long+"=") {
			return true
		}
	}
	return false
}

func apiErrorEnvelope(command string, prop *model.Proposal, err error) model.Envelope {
	if apiErr, ok := err.(*api.APIError); ok {
		return errorEnvelope(command, prop, "api_error", apiErr.Error())
	}
	return errorEnvelope(command, prop, "request_error", err.Error())
}
