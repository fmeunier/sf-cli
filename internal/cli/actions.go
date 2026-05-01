package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"sf-cli/internal/api"
	"sf-cli/internal/model"
)

const (
	actionTypeTicketComment = "ticket_comment"
	actionTypeTicketLabels  = "ticket_labels"
)

type actionsValidateConfig struct {
	ActionFile string
}

type actionsFile struct {
	Actions []intentAction `json:"actions"`
}

type intentAction struct {
	Type    string   `json:"type"`
	Project string   `json:"project"`
	Tracker string   `json:"tracker"`
	Ticket  int      `json:"ticket"`
	Body    string   `json:"body"`
	Labels  []string `json:"labels"`
}

type actionsValidateResult struct {
	OK               bool              `json:"ok"`
	ValidatedActions []validatedAction `json:"validated_actions"`
}

type validatedAction struct {
	Index                int               `json:"index"`
	Type                 string            `json:"type"`
	Target               map[string]any    `json:"target,omitempty"`
	Action               map[string]any    `json:"action,omitempty"`
	CanonicalIdentifiers map[string]any    `json:"canonical_identifiers,omitempty"`
	OK                   bool              `json:"ok"`
	Issues               []validationIssue `json:"issues,omitempty"`
}

type validationIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

func handleActions(ctx context.Context, client *api.Client, args []string) model.Envelope {
	if len(args) == 0 {
		return errorEnvelope("actions", proposal("actions", "dispatch_actions_command", nil, nil), "invalid_arguments", "missing actions subcommand\n\n"+actionsUsage())
	}

	switch args[0] {
	case "validate":
		return runActionsValidate(ctx, client, args[1:])
	default:
		command := "actions." + args[0]
		return errorEnvelope(command, proposal(command, actionForActions(args[0]), nil, nil), "not_implemented", fmt.Sprintf("command %q is not implemented yet\n\n%s", command, actionsUsage()))
	}
}

func runActionsValidate(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseActionsValidateFlags(args)
	command := "actions.validate"
	prop := proposal(command, "validate_actions_file", nil, map[string]any{"file": config.ActionFile})
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	input, err := readActionsFile(config.ActionFile)
	if err != nil {
		return errorEnvelope(command, prop, "invalid_input", err.Error())
	}

	result, err := validateIntentActions(ctx, client, input.Actions)
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

func parseActionsValidateFlags(args []string) (actionsValidateConfig, error) {
	fs := flag.NewFlagSet("actions validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return actionsValidateConfig{}, normalizeFlagError(err)
	}
	if fs.NArg() != 1 {
		return actionsValidateConfig{}, fmt.Errorf("missing required actions file\n\n%s", actionsValidateUsage())
	}

	return actionsValidateConfig{ActionFile: strings.TrimSpace(fs.Arg(0))}, nil
}

func readActionsFile(filePath string) (actionsFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return actionsFile{}, fmt.Errorf("read actions file: %w", err)
	}

	var input actionsFile
	if err := json.Unmarshal(content, &input); err != nil {
		return actionsFile{}, fmt.Errorf("decode actions file: %w", err)
	}
	if len(input.Actions) == 0 {
		return actionsFile{}, fmt.Errorf("actions file must contain at least one action")
	}

	return input, nil
}

func validateIntentActions(ctx context.Context, client *api.Client, actions []intentAction) (actionsValidateResult, error) {
	validated := make([]validatedAction, 0, len(actions))
	allOK := true

	for i, action := range actions {
		validatedAction, err := validateIntentAction(ctx, client, i, action)
		if err != nil {
			return actionsValidateResult{}, err
		}
		if !validatedAction.OK {
			allOK = false
		}
		validated = append(validated, validatedAction)
	}

	return actionsValidateResult{OK: allOK, ValidatedActions: validated}, nil
}

func validateIntentAction(ctx context.Context, client *api.Client, index int, action intentAction) (validatedAction, error) {
	trimmedProject := strings.TrimSpace(action.Project)
	trimmedTracker := strings.TrimSpace(action.Tracker)
	trimmedType := strings.TrimSpace(action.Type)
	validated := validatedAction{
		Index: index,
		Type:  action.Type,
		Target: map[string]any{
			"project": action.Project,
			"tracker": action.Tracker,
			"ticket":  action.Ticket,
		},
		OK: true,
	}
	if trimmedType == actionTypeTicketComment {
		validated.Action = normalizeTicketCommentAction(action)
		validated.CanonicalIdentifiers = ticketCommentCanonicalIdentifiers(trimmedProject, trimmedTracker, action.Ticket, "")
	}
	if trimmedType == actionTypeTicketLabels {
		validated.Action = normalizeTicketLabelsAction(action)
		validated.CanonicalIdentifiers = ticketLabelsCanonicalIdentifiers(trimmedProject, trimmedTracker, action.Ticket)
	}

	if trimmedType != actionTypeTicketComment && trimmedType != actionTypeTicketLabels {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{
			Severity: "error",
			Code:     "unsupported_action_type",
			Field:    "type",
			Message:  fmt.Sprintf("unsupported action type %q", action.Type),
		})
		return validated, nil
	}

	if trimmedProject == "" {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "missing_project", Field: "project", Message: "project is required"})
	}
	if trimmedTracker == "" {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "missing_tracker", Field: "tracker", Message: "tracker is required"})
	}
	if action.Ticket <= 0 {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "invalid_ticket", Field: "ticket", Message: "ticket must be > 0"})
	}

	switch trimmedType {
	case actionTypeTicketComment:
		bodyLength := len(strings.TrimSpace(action.Body))
		if bodyLength == 0 {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "empty_body", Field: "body", Message: "body must not be empty"})
		}
	case actionTypeTicketLabels:
		if len(action.Labels) == 0 {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "missing_labels", Field: "labels", Message: "labels must contain at least one value"})
			break
		}
		for i, label := range action.Labels {
			trimmedLabel := strings.TrimSpace(label)
			field := fmt.Sprintf("labels[%d]", i)
			if trimmedLabel == "" {
				validated.OK = false
				validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "empty_label", Field: field, Message: "labels must not contain empty values"})
				continue
			}
			if strings.Contains(trimmedLabel, ",") {
				validated.OK = false
				validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "unsupported_label_value", Field: field, Message: "labels must not contain commas"})
			}
		}
	}

	if !validated.OK {
		return validated, nil
	}

	project, err := client.GetProject(ctx, trimmedProject)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 404 {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "project_not_found", Field: "project", Message: fmt.Sprintf("project %q was not found", trimmedProject)})
			return validated, nil
		}
		return validatedAction{}, err
	}

	if !projectHasTracker(project.Tools, trimmedTracker) {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "tracker_not_found", Field: "tracker", Message: fmt.Sprintf("tracker %q was not found in project %q", trimmedTracker, trimmedProject)})
		return validated, nil
	}

	ticket, err := client.GetTicket(ctx, api.GetTicketParams{Project: trimmedProject, Tracker: trimmedTracker, TicketID: action.Ticket})
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 404 {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "ticket_not_found", Field: "ticket", Message: fmt.Sprintf("ticket %d was not found", action.Ticket)})
			return validated, nil
		}
		return validatedAction{}, err
	}
	validated.CanonicalIdentifiers = ticketCommentCanonicalIdentifiers(project.Shortname, trimmedTracker, ticket.Ticket.TicketNum, ticket.Ticket.DiscussionThread.ID)
	if trimmedType == actionTypeTicketLabels {
		validated.CanonicalIdentifiers = ticketLabelsCanonicalIdentifiers(project.Shortname, trimmedTracker, ticket.Ticket.TicketNum)
	} else {
		if ticket.Ticket.DiscussionDisabled {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "ticket_discussion_disabled", Field: "ticket", Message: fmt.Sprintf("ticket %d does not accept discussion posts", action.Ticket)})
			return validated, nil
		}
		if strings.TrimSpace(ticket.Ticket.DiscussionThread.ID) == "" {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "discussion_thread_unavailable", Field: "ticket", Message: fmt.Sprintf("ticket %d does not expose a discussion thread id for posting comments", action.Ticket)})
			return validated, nil
		}
	}

	return validated, nil
}

func normalizeTicketCommentAction(action intentAction) map[string]any {
	return map[string]any{
		"type": actionTypeTicketComment,
		"target": map[string]any{
			"project": strings.TrimSpace(action.Project),
			"tracker": strings.TrimSpace(action.Tracker),
			"ticket":  action.Ticket,
		},
		"inputs": map[string]any{
			"body": action.Body,
		},
	}
}

func normalizeTicketLabelsAction(action intentAction) map[string]any {
	labels := make([]string, 0, len(action.Labels))
	for _, label := range action.Labels {
		labels = append(labels, strings.TrimSpace(label))
	}

	return map[string]any{
		"type": actionTypeTicketLabels,
		"target": map[string]any{
			"project": strings.TrimSpace(action.Project),
			"tracker": strings.TrimSpace(action.Tracker),
			"ticket":  action.Ticket,
		},
		"inputs": map[string]any{
			"labels": labels,
		},
	}
}

func ticketCommentCanonicalIdentifiers(project string, tracker string, ticket int, discussionThreadID string) map[string]any {
	canonical := make(map[string]any)
	if project != "" {
		canonical["project"] = project
	}
	if tracker != "" {
		canonical["tracker"] = tracker
	}
	if ticket > 0 {
		canonical["ticket_num"] = ticket
	}
	if discussionThreadID != "" {
		canonical["discussion_thread_id"] = discussionThreadID
	}
	if len(canonical) == 0 {
		return nil
	}
	return canonical
}

func ticketLabelsCanonicalIdentifiers(project string, tracker string, ticket int) map[string]any {
	return ticketCommentCanonicalIdentifiers(project, tracker, ticket, "")
}

func projectHasTracker(tools []api.ProjectTool, tracker string) bool {
	for _, tool := range tools {
		if tool.MountPoint == tracker {
			return true
		}
	}
	return false
}
