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
	Fields  []string
	All     bool
}

const openTicketsActivityQuery = "!status:closed-rejected && !status:closed-invalid && !status:closed-duplicate && !status:closed-out-of-date && !status:closed-accepted && !status:closed-works-for-me && !status:closed && !status:closed-wont-fix && !status:closed-fixed"

type ticketsSearchConfig struct {
	Project string
	Tracker string
	Query   string
	Cursor  string
	Limit   int
	Fields  []string
}

type ticketsGetConfig struct {
	Project string
	Tracker string
	Ticket  int
	Fields  []string
}

var ticketProjectionFields = map[string]func(api.Ticket) any{
	"assigned_to": func(ticket api.Ticket) any { return ticket.AssignedTo },
	"attachments": func(ticket api.Ticket) any { return ticket.Attachments },
	"created_at":  func(ticket api.Ticket) any { return ticket.CreatedDate },
	"custom_fields": func(ticket api.Ticket) any {
		return ticket.CustomFields
	},
	"description":         func(ticket api.Ticket) any { return ticket.Description },
	"discussion_disabled": func(ticket api.Ticket) any { return ticket.DiscussionDisabled },
	"id":                  func(ticket api.Ticket) any { return ticket.TicketNum },
	"labels":              func(ticket api.Ticket) any { return ticket.Labels },
	"private":             func(ticket api.Ticket) any { return ticket.Private },
	"related_artifacts":   func(ticket api.Ticket) any { return ticket.RelatedArtifacts },
	"reported_by":         func(ticket api.Ticket) any { return ticket.ReportedBy },
	"status":              func(ticket api.Ticket) any { return ticket.Status },
	"title":               func(ticket api.Ticket) any { return ticket.Summary },
	"updated_at":          func(ticket api.Ticket) any { return ticket.ModDate },
}

var listTicketProjectionFields = []string{"id", "title", "status", "reported_by", "assigned_to", "labels", "created_at", "updated_at"}

var getTicketProjectionFields = []string{"id", "title", "description", "status", "reported_by", "assigned_to", "labels", "private", "discussion_disabled", "custom_fields", "attachments", "related_artifacts", "created_at", "updated_at"}

var commentProjectionFields = []string{"id", "author", "body", "created_at", "edited_at", "subject", "type", "is_meta", "attachments"}

var activityProjectionFields = []string{"id", "title", "status", "activity_type", "updated_at", "last_comment_at", "last_comment_author"}

var commentFieldProjectors = map[string]func(api.Comment) any{
	"attachments": func(comment api.Comment) any { return comment.Attachments },
	"author":      func(comment api.Comment) any { return comment.Author },
	"body":        func(comment api.Comment) any { return comment.Body },
	"created_at":  func(comment api.Comment) any { return comment.CreatedAt },
	"edited_at":   func(comment api.Comment) any { return comment.EditedAt },
	"id":          func(comment api.Comment) any { return comment.ID },
	"is_meta":     func(comment api.Comment) any { return comment.IsMeta },
	"subject":     func(comment api.Comment) any { return comment.Subject },
	"type":        func(comment api.Comment) any { return comment.Type },
}

var activityFieldProjectors = map[string]func(ticketActivity) any{
	"activity_type":       func(activity ticketActivity) any { return activity.ActivityType },
	"id":                  func(activity ticketActivity) any { return activity.TicketNum },
	"last_comment_at":     func(activity ticketActivity) any { return activity.LastCommentAt },
	"last_comment_author": func(activity ticketActivity) any { return activity.LastCommentAuthor },
	"status":              func(activity ticketActivity) any { return activity.Status },
	"title":               func(activity ticketActivity) any { return activity.Summary },
	"updated_at":          func(activity ticketActivity) any { return activity.UpdatedAt },
}

func runTicketsList(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsListFlags(args)
	command := "tickets.list"
	prop := proposal(command, "list_tickets", map[string]any{"project": config.Project, "tracker": config.Tracker}, ticketsListInputs(config))
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
	if len(config.Fields) != 0 {
		return successEnvelope(command, prop, projectTicketListResult(result, config.Fields))
	}

	return successEnvelope(command, prop, projectCanonicalTicketListResult(result))
}

func runTicketsSearch(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsSearchFlags(args)
	command := "tickets.search"
	prop := proposal(command, "search_tickets", map[string]any{"project": config.Project, "tracker": config.Tracker}, ticketsSearchInputs(config))
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.SearchTickets(ctx, api.SearchTicketsParams{
		Project: config.Project,
		Tracker: config.Tracker,
		Query:   config.Query,
		Sort:    "",
		Cursor:  config.Cursor,
		Limit:   config.Limit,
	})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}
	if len(config.Fields) != 0 {
		return successEnvelope(command, prop, projectTicketSearchResult(result, config.Fields))
	}

	return successEnvelope(command, prop, projectCanonicalTicketSearchResult(result))
}

func runTicketsGet(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsGetFlags("tickets get", args, getTicketProjectionFields)
	command := "tickets.get"
	prop := proposal(command, "get_ticket", map[string]any{"project": config.Project, "tracker": config.Tracker, "ticket": config.Ticket}, ticketsGetInputs(config))
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetTicket(ctx, api.GetTicketParams{Project: config.Project, Tracker: config.Tracker, TicketID: config.Ticket})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}
	if len(config.Fields) != 0 {
		return successEnvelope(command, prop, map[string]any{"ticket": projectTicket(result.Ticket, config.Fields)})
	}

	return successEnvelope(command, prop, projectCanonicalTicketDetailResult(result))
}

func runTicketsComments(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTicketsCommentsFlags(args)
	command := "tickets.comments"
	prop := proposal(command, "get_ticket_comments", map[string]any{"project": config.Project, "tracker": config.Tracker, "ticket": config.Ticket}, ticketsGetInputs(config))
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetTicketComments(ctx, api.GetTicketParams{Project: config.Project, Tracker: config.Tracker, TicketID: config.Ticket})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}
	if len(config.Fields) != 0 {
		return successEnvelope(command, prop, projectCommentsResult(result, config.Fields))
	}

	return successEnvelope(command, prop, result)
}

type ticketActivity struct {
	TicketNum         int    `json:"ticket_num"`
	Summary           string `json:"summary"`
	Status            string `json:"status"`
	ActivityType      string `json:"activity_type"`
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
	config, err := parseTicketsActivityFlags(args)
	command := "tickets.activity"
	prop := proposal(command, "list_ticket_activity", map[string]any{"project": config.Project, "tracker": config.Tracker}, ticketsActivityInputs(config))
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	query := openTicketsActivityQuery
	if config.All {
		query = "*"
	}

	listResult, err := client.SearchTickets(ctx, api.SearchTicketsParams{
		Project: config.Project,
		Tracker: config.Tracker,
		Query:   query,
		Sort:    "mod_date_dt desc",
		Cursor:  config.Cursor,
		Limit:   config.Limit,
	})
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	activities := make([]ticketActivity, len(listResult.Tickets))
	for i, ticket := range listResult.Tickets {
		activities[i] = ticketActivity{
			TicketNum:    ticket.TicketNum,
			Summary:      ticket.Summary,
			Status:       ticket.Status,
			ActivityType: "ticket",
			UpdatedAt:    ticket.ModDate,
		}
	}

	sort.SliceStable(activities, func(i int, j int) bool {
		left := activities[i]
		right := activities[j]
		if left.UpdatedAt != right.UpdatedAt {
			return left.UpdatedAt > right.UpdatedAt
		}
		return left.TicketNum > right.TicketNum
	})

	result := ticketsActivityResponse{
		Tickets:    activities,
		Count:      listResult.Count,
		Limit:      listResult.Limit,
		Pagination: listResult.Pagination,
	}
	if len(config.Fields) != 0 {
		return successEnvelope(command, prop, projectActivityResult(result, config.Fields))
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
	var rawFields string
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.StringVar(&config.Cursor, "cursor", "", "Opaque cursor for the next page")
	fs.StringVar(&rawFields, "fields", "", "Comma-separated projected ticket fields")
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
	fields, err := parseFields(rawFields, hasFlag(args, "fields"), listTicketProjectionFields, "tickets list")
	if err != nil {
		return ticketsListConfig{}, err
	}
	config.Fields = fields

	return config, nil
}

func parseTicketsSearchFlags(args []string) (ticketsSearchConfig, error) {
	fs := flag.NewFlagSet("tickets search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := ticketsSearchConfig{}
	var rawFields string
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.StringVar(&config.Query, "query", "", "Ticket search query")
	fs.StringVar(&config.Cursor, "cursor", "", "Opaque cursor for the next page")
	fs.StringVar(&rawFields, "fields", "", "Comma-separated projected ticket fields")
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
	fields, err := parseFields(rawFields, hasFlag(args, "fields"), listTicketProjectionFields, "tickets search")
	if err != nil {
		return ticketsSearchConfig{}, err
	}
	config.Fields = fields
	return config, nil
}

func parseTicketsGetFlags(name string, args []string, allowedFields []string) (ticketsGetConfig, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := ticketsGetConfig{}
	var rawFields string
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.IntVar(&config.Ticket, "ticket", 0, "Ticket number")
	fs.StringVar(&rawFields, "fields", "", "Comma-separated projected ticket fields")

	if err := fs.Parse(args); err != nil {
		return ticketsGetConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return ticketsGetConfig{}, err
	}
	if config.Ticket <= 0 {
		return ticketsGetConfig{}, fmt.Errorf("--ticket must be > 0")
	}
	fields, err := parseFields(rawFields, hasFlag(args, "fields"), allowedFields, name)
	if err != nil {
		return ticketsGetConfig{}, err
	}
	config.Fields = fields

	return config, nil
}

func parseTicketsCommentsFlags(args []string) (ticketsGetConfig, error) {
	return parseTicketsGetFlags("tickets comments", args, commentProjectionFields)
}

func parseTicketsActivityFlags(args []string) (ticketsListConfig, error) {
	if hasFlag(args, "query") {
		return ticketsListConfig{}, fmt.Errorf("--query is only supported by `tickets search`; use `sf tickets search --project <project> --tracker <tracker> --query <query>`")
	}

	fs := flag.NewFlagSet("tickets activity", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := ticketsListConfig{}
	var rawFields string
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.StringVar(&config.Cursor, "cursor", "", "Opaque cursor for the next page")
	fs.StringVar(&rawFields, "fields", "", "Comma-separated projected ticket activity fields")
	fs.IntVar(&config.Limit, "limit", 25, "Page size")
	fs.BoolVar(&config.All, "all", false, "Include closed issues")

	if err := fs.Parse(args); err != nil {
		return ticketsListConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return ticketsListConfig{}, err
	}
	if err := validatePagination(config.Cursor, config.Limit); err != nil {
		return ticketsListConfig{}, err
	}
	fields, err := parseFields(rawFields, hasFlag(args, "fields"), activityProjectionFields, "tickets activity")
	if err != nil {
		return ticketsListConfig{}, err
	}
	config.Fields = fields

	return config, nil
}

func ticketsListInputs(config ticketsListConfig) map[string]any {
	inputs := map[string]any{"cursor": config.Cursor, "limit": config.Limit}
	if len(config.Fields) != 0 {
		inputs["fields"] = config.Fields
	}
	return inputs
}

func ticketsActivityInputs(config ticketsListConfig) map[string]any {
	inputs := ticketsListInputs(config)
	if config.All {
		inputs["all"] = true
	}
	return inputs
}

func ticketsSearchInputs(config ticketsSearchConfig) map[string]any {
	inputs := map[string]any{"query": config.Query, "cursor": config.Cursor, "limit": config.Limit}
	if len(config.Fields) != 0 {
		inputs["fields"] = config.Fields
	}
	return inputs
}

func ticketsGetInputs(config ticketsGetConfig) map[string]any {
	if len(config.Fields) == 0 {
		return nil
	}
	return map[string]any{"fields": config.Fields}
}

func projectTicket(ticket api.Ticket, fields []string) map[string]any {
	projected := make(map[string]any, len(fields))
	for _, field := range fields {
		projected[field] = ticketProjectionFields[field](ticket)
	}
	return projected
}

func projectTicketListResult(result api.TicketListResponse, fields []string) map[string]any {
	projectedTickets := make([]map[string]any, 0, len(result.Tickets))
	for _, ticket := range result.Tickets {
		projectedTickets = append(projectedTickets, projectTicket(ticket, fields))
	}

	projected := map[string]any{
		"tickets":    projectedTickets,
		"count":      result.Count,
		"limit":      result.Limit,
		"pagination": result.Pagination,
	}
	if len(result.Milestones) != 0 {
		projected["milestones"] = result.Milestones
	}
	return projected
}

func projectCanonicalTicketListResult(result api.TicketListResponse) map[string]any {
	projectedTickets := make([]map[string]any, 0, len(result.Tickets))
	for _, ticket := range result.Tickets {
		projectedTickets = append(projectedTickets, projectCanonicalTicket(ticket, canonicalTicketListCommand))
	}

	projected := map[string]any{
		"tickets":    projectedTickets,
		"count":      result.Count,
		"limit":      result.Limit,
		"pagination": result.Pagination,
	}
	if len(result.Milestones) != 0 {
		projected["milestones"] = result.Milestones
	}
	return projected
}

func projectTicketSearchResult(result api.TicketSearchResponse, fields []string) map[string]any {
	projectedTickets := make([]map[string]any, 0, len(result.Tickets))
	for _, ticket := range result.Tickets {
		projectedTickets = append(projectedTickets, projectTicket(ticket, fields))
	}

	projected := map[string]any{
		"tickets":    projectedTickets,
		"count":      result.Count,
		"limit":      result.Limit,
		"pagination": result.Pagination,
	}
	if result.Sort != "" {
		projected["sort"] = result.Sort
	}
	if len(result.FilterChoices) != 0 {
		projected["filter_choices"] = result.FilterChoices
	}
	return projected
}

func projectCanonicalTicketSearchResult(result api.TicketSearchResponse) map[string]any {
	projectedTickets := make([]map[string]any, 0, len(result.Tickets))
	for _, ticket := range result.Tickets {
		projectedTickets = append(projectedTickets, projectCanonicalTicket(ticket, canonicalTicketSearchCommand))
	}

	projected := map[string]any{
		"tickets":    projectedTickets,
		"count":      result.Count,
		"limit":      result.Limit,
		"pagination": result.Pagination,
	}
	if result.Sort != "" {
		projected["sort"] = result.Sort
	}
	if len(result.FilterChoices) != 0 {
		projected["filter_choices"] = result.FilterChoices
	}
	return projected
}

func projectCanonicalTicketDetailResult(result api.TicketDetailResponse) map[string]any {
	return map[string]any{"ticket": projectCanonicalTicket(result.Ticket, canonicalTicketGetCommand)}
}

func projectComment(comment api.Comment, fields []string) map[string]any {
	projected := make(map[string]any, len(fields))
	for _, field := range fields {
		projected[field] = commentFieldProjectors[field](comment)
	}
	return projected
}

func projectCommentsResult(result api.TicketCommentsResponse, fields []string) map[string]any {
	projectedComments := make([]map[string]any, 0, len(result.Comments))
	for _, comment := range result.Comments {
		projectedComments = append(projectedComments, projectComment(comment, fields))
	}

	return map[string]any{
		"thread":   result.Thread,
		"comments": projectedComments,
	}
}

func projectActivity(activity ticketActivity, fields []string) map[string]any {
	projected := make(map[string]any, len(fields))
	for _, field := range fields {
		projected[field] = activityFieldProjectors[field](activity)
	}
	return projected
}

func projectActivityResult(result ticketsActivityResponse, fields []string) map[string]any {
	projectedActivities := make([]map[string]any, 0, len(result.Tickets))
	for _, activity := range result.Tickets {
		projectedActivities = append(projectedActivities, projectActivity(activity, fields))
	}

	return map[string]any{
		"tickets":    projectedActivities,
		"count":      result.Count,
		"limit":      result.Limit,
		"pagination": result.Pagination,
	}
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
