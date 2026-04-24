package cli

import "sf-cli/internal/api"

const (
	canonicalTicketListCommand   = "tickets list"
	canonicalTicketSearchCommand = "tickets search"
	canonicalTicketGetCommand    = "tickets get"
)

// canonicalTicketSchema is the source of truth for ticket payloads emitted by
// tickets list/search/get when callers do not request a compact --fields
// projection.
//
// Compatibility rules:
//   - Preserve existing field names, meanings, and nullability.
//   - Additive changes must start as optional fields.
//   - Promote a field to additional commands only after updating this contract,
//     README output docs, and the conformance tests.
//   - Optional fields are omitted when the upstream tracker does not supply a
//     meaningful value; ticket payloads do not use JSON null today.
var canonicalTicketSchema = []ticketSchemaField{
	{name: "ticket_num", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, required: true, project: func(ticket api.Ticket) any { return ticket.TicketNum }},
	{name: "summary", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, required: true, project: func(ticket api.Ticket) any { return ticket.Summary }},
	{name: "status", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, required: true, project: func(ticket api.Ticket) any { return ticket.Status }},
	{name: "reported_by", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.ReportedBy }, include: func(ticket api.Ticket) bool { return ticket.ReportedBy != "" }},
	{name: "assigned_to", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.AssignedTo }, include: func(ticket api.Ticket) bool { return ticket.AssignedTo != "" }},
	{name: "labels", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.Labels }, include: func(ticket api.Ticket) bool { return len(ticket.Labels) != 0 }},
	{name: "created_date", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.CreatedDate }, include: func(ticket api.Ticket) bool { return ticket.CreatedDate != "" }},
	{name: "mod_date", commands: []string{canonicalTicketListCommand, canonicalTicketSearchCommand, canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.ModDate }, include: func(ticket api.Ticket) bool { return ticket.ModDate != "" }},
	{name: "description", commands: []string{canonicalTicketGetCommand}, required: true, project: func(ticket api.Ticket) any { return ticket.Description }},
	{name: "private", commands: []string{canonicalTicketGetCommand}, required: true, project: func(ticket api.Ticket) any { return ticket.Private }},
	{name: "discussion_disabled", commands: []string{canonicalTicketGetCommand}, required: true, project: func(ticket api.Ticket) any { return ticket.DiscussionDisabled }},
	{name: "discussion_thread", commands: []string{canonicalTicketGetCommand}, required: true, project: func(ticket api.Ticket) any { return ticket.DiscussionThread }},
	{name: "discussion_thread_url", commands: []string{canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.DiscussionThreadURL }, include: func(ticket api.Ticket) bool { return ticket.DiscussionThreadURL != "" }},
	{name: "custom_fields", commands: []string{canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.CustomFields }, include: func(ticket api.Ticket) bool { return len(ticket.CustomFields) != 0 }},
	{name: "attachments", commands: []string{canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.Attachments }, include: func(ticket api.Ticket) bool { return len(ticket.Attachments) != 0 }},
	{name: "related_artifacts", commands: []string{canonicalTicketGetCommand}, optional: true, project: func(ticket api.Ticket) any { return ticket.RelatedArtifacts }, include: func(ticket api.Ticket) bool { return len(ticket.RelatedArtifacts) != 0 }},
}

type ticketSchemaField struct {
	name     string
	commands []string
	required bool
	optional bool
	nullable bool
	project  func(api.Ticket) any
	include  func(api.Ticket) bool
}

func (field ticketSchemaField) supports(command string) bool {
	for _, supported := range field.commands {
		if supported == command {
			return true
		}
	}
	return false
}

func canonicalTicketFieldNames(command string) []string {
	fields := make([]string, 0, len(canonicalTicketSchema))
	for _, field := range canonicalTicketSchema {
		if field.supports(command) {
			fields = append(fields, field.name)
		}
	}
	return fields
}

func projectCanonicalTicket(ticket api.Ticket, command string) map[string]any {
	projected := make(map[string]any)
	for _, field := range canonicalTicketSchema {
		if !field.supports(command) {
			continue
		}
		if field.optional && field.include != nil && !field.include(ticket) {
			continue
		}
		projected[field.name] = field.project(ticket)
	}
	return projected
}
