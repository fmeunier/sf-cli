package api

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
)

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
	Tickets    []Ticket    `json:"tickets"`
	Count      int         `json:"count"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Milestones []Milestone `json:"milestones,omitempty"`
}

type TicketSearchResponse struct {
	Tickets       []Ticket       `json:"tickets"`
	Count         int            `json:"count"`
	Page          int            `json:"page"`
	Limit         int            `json:"limit"`
	Sort          string         `json:"sort,omitempty"`
	FilterChoices map[string]any `json:"filter_choices,omitempty"`
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
	ID           string              `json:"_id"`
	DiscussionID string              `json:"discussion_id,omitempty"`
	Subject      string              `json:"subject,omitempty"`
	Posts        []RawDiscussionPost `json:"posts,omitempty"`
}

type TicketCommentsResponse struct {
	Thread   CommentThread `json:"thread"`
	Comments []Comment     `json:"comments"`
}

type CommentThread struct {
	ID      string `json:"id,omitempty"`
	Subject string `json:"subject,omitempty"`
}

type Comment struct {
	ID          string `json:"id,omitempty"`
	Author      string `json:"author,omitempty"`
	Body        string `json:"body"`
	CreatedAt   string `json:"created_at,omitempty"`
	EditedAt    any    `json:"edited_at,omitempty"`
	Subject     string `json:"subject,omitempty"`
	IsMeta      bool   `json:"is_meta"`
	Attachments []any  `json:"attachments,omitempty"`
}

type rawDiscussionThreadResponse struct {
	Thread rawDiscussionThread `json:"thread"`
}

type rawDiscussionThread struct {
	ID           string              `json:"_id"`
	DiscussionID string              `json:"discussion_id,omitempty"`
	Subject      string              `json:"subject,omitempty"`
	Posts        []RawDiscussionPost `json:"posts,omitempty"`
	Count        int                 `json:"count,omitempty"`
	Page         int                 `json:"page,omitempty"`
	Limit        int                 `json:"limit,omitempty"`
}

type RawDiscussionPost struct {
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

func (c *Client) GetTicketComments(ctx context.Context, params GetTicketParams) (TicketCommentsResponse, error) {
	ticket, err := c.GetTicket(ctx, params)
	if err != nil {
		return TicketCommentsResponse{}, err
	}

	threadID := ticket.Ticket.DiscussionThread.ID
	if threadID == "" {
		return TicketCommentsResponse{Comments: []Comment{}}, nil
	}

	var raw rawDiscussionThreadResponse
	threadPath := fmt.Sprintf("%s/_discuss/thread/%s", trackerPath(params.Project, params.Tracker), threadID)
	if err := c.GetJSON(ctx, threadPath, nil, &raw); err != nil {
		return TicketCommentsResponse{}, err
	}

	return normalizeComments(raw.Thread), nil
}

func normalizeComments(thread rawDiscussionThread) TicketCommentsResponse {
	comments := make([]Comment, 0, len(thread.Posts))
	for _, post := range thread.Posts {
		comments = append(comments, Comment{
			ID:          post.Slug,
			Author:      post.Author,
			Body:        post.Text,
			CreatedAt:   post.Timestamp,
			EditedAt:    post.LastEdited,
			Subject:     post.Subject,
			IsMeta:      post.IsMeta,
			Attachments: post.Attachments,
		})
	}

	sort.SliceStable(comments, func(i int, j int) bool {
		left := comments[i]
		right := comments[j]

		if left.CreatedAt != right.CreatedAt {
			if left.CreatedAt == "" {
				return false
			}
			if right.CreatedAt == "" {
				return true
			}
			return left.CreatedAt < right.CreatedAt
		}
		return left.ID < right.ID
	})

	return TicketCommentsResponse{
		Thread: CommentThread{
			ID:      thread.ID,
			Subject: thread.Subject,
		},
		Comments: comments,
	}
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
