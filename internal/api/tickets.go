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
