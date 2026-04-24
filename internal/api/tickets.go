package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

type TicketSummary struct {
	TicketNum int    `json:"ticket_num"`
	Summary   string `json:"summary"`
}

type Milestone struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
	Default     any    `json:"default"`
	Complete    bool   `json:"complete"`
	Closed      int    `json:"closed"`
	Total       int    `json:"total"`
}

type TicketListResponse struct {
	Tickets    []TicketSummary `json:"tickets"`
	Count      int             `json:"count"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	Milestones []Milestone     `json:"milestones,omitempty"`
}

type TicketSearchResponse struct {
	Tickets       []TicketSummary `json:"tickets"`
	Count         int             `json:"count"`
	Page          int             `json:"page"`
	Limit         int             `json:"limit"`
	Sort          string          `json:"sort,omitempty"`
	FilterChoices map[string]any  `json:"filter_choices,omitempty"`
}

type ListTicketsParams struct {
	Project string
	Tracker string
	Page    int
	Limit   int
}

type SearchTicketsParams struct {
	Project string
	Tracker string
	Query   string
	Page    int
	Limit   int
}

type GetTicketParams struct {
	Project  string
	Tracker  string
	TicketID int
}

type TicketDetailResponse struct {
	Ticket Ticket `json:"ticket"`
}

type Ticket struct {
	TicketNum           int                 `json:"ticket_num"`
	Summary             string              `json:"summary"`
	Description         string              `json:"description"`
	Status              string              `json:"status"`
	ReportedBy          string              `json:"reported_by,omitempty"`
	AssignedTo          string              `json:"assigned_to,omitempty"`
	Labels              []string            `json:"labels,omitempty"`
	Private             bool                `json:"private"`
	DiscussionDisabled  bool                `json:"discussion_disabled"`
	DiscussionThread    DiscussionThreadRef `json:"discussion_thread"`
	DiscussionThreadURL string              `json:"discussion_thread_url,omitempty"`
	CustomFields        map[string]any      `json:"custom_fields,omitempty"`
	Attachments         []any               `json:"attachments,omitempty"`
	RelatedArtifacts    []any               `json:"related_artifacts,omitempty"`
	CreatedDate         string              `json:"created_date,omitempty"`
	ModDate             string              `json:"mod_date,omitempty"`
}

type DiscussionThreadRef struct {
	ID           string           `json:"_id"`
	DiscussionID string           `json:"discussion_id,omitempty"`
	Subject      string           `json:"subject,omitempty"`
	Posts        []DiscussionPost `json:"posts,omitempty"`
}

type DiscussionThreadResponse struct {
	Thread DiscussionThread `json:"thread"`
}

type DiscussionThread struct {
	ID           string           `json:"_id"`
	DiscussionID string           `json:"discussion_id,omitempty"`
	Subject      string           `json:"subject,omitempty"`
	Posts        []DiscussionPost `json:"posts,omitempty"`
	Count        int              `json:"count,omitempty"`
	Page         int              `json:"page,omitempty"`
	Limit        int              `json:"limit,omitempty"`
}

type DiscussionPost struct {
	Text        string `json:"text"`
	IsMeta      bool   `json:"is_meta"`
	Author      string `json:"author,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	LastEdited  any    `json:"last_edited,omitempty"`
	Slug        string `json:"slug,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Attachments []any  `json:"attachments,omitempty"`
}

func (c *Client) ListTickets(ctx context.Context, params ListTicketsParams) (TicketListResponse, error) {
	var out TicketListResponse
	if err := c.GetJSON(ctx, trackerPath(params.Project, params.Tracker), paginationQuery(params.Page, params.Limit), &out); err != nil {
		return TicketListResponse{}, err
	}
	return out, nil
}

func (c *Client) SearchTickets(ctx context.Context, params SearchTicketsParams) (TicketSearchResponse, error) {
	query := paginationQuery(params.Page, params.Limit)
	query.Set("q", params.Query)

	var out TicketSearchResponse
	if err := c.GetJSON(ctx, trackerPath(params.Project, params.Tracker)+"/search", query, &out); err != nil {
		return TicketSearchResponse{}, err
	}
	return out, nil
}

func (c *Client) GetTicket(ctx context.Context, params GetTicketParams) (TicketDetailResponse, error) {
	var out TicketDetailResponse
	if err := c.GetJSON(ctx, fmt.Sprintf("%s/%d", trackerPath(params.Project, params.Tracker), params.TicketID), nil, &out); err != nil {
		return TicketDetailResponse{}, err
	}
	return out, nil
}

func (c *Client) GetTicketComments(ctx context.Context, params GetTicketParams) (DiscussionThreadResponse, error) {
	ticket, err := c.GetTicket(ctx, params)
	if err != nil {
		return DiscussionThreadResponse{}, err
	}

	threadID := ticket.Ticket.DiscussionThread.ID
	if threadID == "" {
		return DiscussionThreadResponse{Thread: DiscussionThread{Posts: []DiscussionPost{}}}, nil
	}

	var out DiscussionThreadResponse
	threadPath := fmt.Sprintf("%s/_discuss/thread/%s", trackerPath(params.Project, params.Tracker), threadID)
	if err := c.GetJSON(ctx, threadPath, nil, &out); err != nil {
		return DiscussionThreadResponse{}, err
	}
	return out, nil
}

func paginationQuery(page int, limit int) url.Values {
	query := url.Values{}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	return query
}

func trackerPath(project string, tracker string) string {
	return fmt.Sprintf("p/%s/%s", project, tracker)
}
