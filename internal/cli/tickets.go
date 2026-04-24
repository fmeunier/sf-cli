package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"sf-cli/internal/api"
	"sf-cli/internal/model"
)

type ticketsListConfig struct {
	Project string
	Tracker string
	Cursor  string
	Limit   int
}

type ticketsSearchConfig struct {
	Project string
	Tracker string
	Query   string
	Cursor  string
	Limit   int
}

type ticketsGetConfig struct {
	Project string
	Tracker string
	Ticket  int
}

func runTicketsList(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsListFlags(args)
	command := "tickets.list"
	prop := proposal(command, "list_tickets", map[string]any{"project": config.Project, "tracker": config.Tracker}, map[string]any{"cursor": config.Cursor, "limit": config.Limit})
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.ListTickets(ctx, api.ListTicketsParams{
		Project: config.Project,
		Tracker: config.Tracker,
		Cursor:  config.Cursor,
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
	prop := proposal(command, "search_tickets", map[string]any{"project": config.Project, "tracker": config.Tracker}, map[string]any{"query": config.Query, "cursor": config.Cursor, "limit": config.Limit})
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.SearchTickets(ctx, api.SearchTicketsParams{
		Project: config.Project,
		Tracker: config.Tracker,
		Query:   config.Query,
		Cursor:  config.Cursor,
		Limit:   config.Limit,
	})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

func runTicketsGet(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsGetFlags("tickets get", args)
	command := "tickets.get"
	prop := proposal(command, "get_ticket", map[string]any{"project": config.Project, "tracker": config.Tracker, "ticket": config.Ticket}, nil)
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetTicket(ctx, api.GetTicketParams{Project: config.Project, Tracker: config.Tracker, TicketID: config.Ticket})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

func runTicketsComments(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsGetFlags("tickets comments", args)
	command := "tickets.comments"
	prop := proposal(command, "get_ticket_comments", map[string]any{"project": config.Project, "tracker": config.Tracker, "ticket": config.Ticket}, nil)
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetTicketComments(ctx, api.GetTicketParams{Project: config.Project, Tracker: config.Tracker, TicketID: config.Ticket})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

type ticketActivity struct {
	TicketNum         int    `json:"ticket_num"`
	Summary           string `json:"summary"`
	Status            string `json:"status"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	LastCommentAt     string `json:"last_comment_at,omitempty"`
	LastCommentAuthor string `json:"last_comment_author,omitempty"`
}

type ticketsActivityResponse struct {
	Tickets    []ticketActivity `json:"tickets"`
	Count      int              `json:"count"`
	Limit      int              `json:"limit"`
	Pagination api.Pagination   `json:"pagination"`
}

func runTicketsActivity(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsListFlags(args)
	command := "tickets.activity"
	prop := proposal(command, "list_ticket_activity", map[string]any{"project": config.Project, "tracker": config.Tracker}, map[string]any{"cursor": config.Cursor, "limit": config.Limit})
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	listResult, err := client.ListTickets(ctx, api.ListTicketsParams{
		Project: config.Project,
		Tracker: config.Tracker,
		Cursor:  config.Cursor,
		Limit:   config.Limit,
	})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	activities := make([]ticketActivity, 0, len(listResult.Tickets))
	for _, ticket := range listResult.Tickets {
		activity := ticketActivity{
			TicketNum: ticket.TicketNum,
			Summary:   ticket.Summary,
			Status:    ticket.Status,
			UpdatedAt: ticket.ModDate,
		}

		commentsResult, err := client.GetTicketComments(ctx, api.GetTicketParams{Project: config.Project, Tracker: config.Tracker, TicketID: ticket.TicketNum})
		if err != nil {
			return apiErrorEnvelope(command, prop, err)
		}
		if lastComment, ok := lastComment(commentsResult.Comments); ok {
			activity.LastCommentAt = lastComment.CreatedAt
			activity.LastCommentAuthor = lastComment.Author
			if activity.UpdatedAt == "" || lastComment.CreatedAt > activity.UpdatedAt {
				activity.UpdatedAt = lastComment.CreatedAt
			}
		}

		activities = append(activities, activity)
	}

	sort.SliceStable(activities, func(i int, j int) bool {
		left := activities[i]
		right := activities[j]
		if left.UpdatedAt != right.UpdatedAt {
			return left.UpdatedAt > right.UpdatedAt
		}
		return left.TicketNum > right.TicketNum
	})

	return successEnvelope(command, prop, ticketsActivityResponse{
		Tickets:    activities,
		Count:      listResult.Count,
		Limit:      listResult.Limit,
		Pagination: listResult.Pagination,
	})
}

func lastComment(comments []api.Comment) (api.Comment, bool) {
	for i := len(comments) - 1; i >= 0; i-- {
		if comments[i].CreatedAt != "" || comments[i].Author != "" || comments[i].Body != "" {
			return comments[i], true
		}
	}
	return api.Comment{}, false
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
	fs.StringVar(&config.Cursor, "cursor", "", "Opaque cursor for the next page")
	fs.IntVar(&config.Limit, "limit", 25, "Page size")

	if err := fs.Parse(args); err != nil {
		return ticketsListConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return ticketsListConfig{}, err
	}
	if err := validatePagination(config.Cursor, config.Limit); err != nil {
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
	fs.StringVar(&config.Cursor, "cursor", "", "Opaque cursor for the next page")
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
	if err := validatePagination(config.Cursor, config.Limit); err != nil {
		return ticketsSearchConfig{}, err
	}

	config.Query = strings.TrimSpace(config.Query)
	return config, nil
}

func parseTicketsGetFlags(name string, args []string) (ticketsGetConfig, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := ticketsGetConfig{}
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.IntVar(&config.Ticket, "ticket", 0, "Ticket number")

	if err := fs.Parse(args); err != nil {
		return ticketsGetConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return ticketsGetConfig{}, err
	}
	if config.Ticket <= 0 {
		return ticketsGetConfig{}, fmt.Errorf("--ticket must be > 0")
	}

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

func validatePagination(cursor string, limit int) error {
	if _, err := apiCursorToPage(cursor); err != nil {
		return err
	}
	if limit <= 0 {
		return fmt.Errorf("--limit must be > 0")
	}
	return nil
}

func apiCursorToPage(cursor string) (int, error) {
	trimmed := strings.TrimSpace(cursor)
	if trimmed == "" {
		return 0, nil
	}

	page, err := strconv.Atoi(trimmed)
	if err != nil || page < 0 {
		return 0, fmt.Errorf("--cursor must be an opaque numeric token returned by this CLI")
	}

	return page, nil
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
