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
	actionTypeTicketCreate  = "ticket_create"
	actionTypeTicketComment = "ticket_comment"
	actionTypeTicketLabels  = "ticket_labels"
)

type actionsValidateConfig struct {
	ActionFile string
}

type actionsApplyConfig struct {
	ActionFile string
	Confirm    bool
}

type actionsFile struct {
	Actions []intentAction `json:"actions"`
}

type intentAction struct {
	Type               string         `json:"type"`
	Project            string         `json:"project"`
	Tracker            string         `json:"tracker"`
	Ticket             int            `json:"ticket"`
	Summary            string         `json:"summary"`
	Description        string         `json:"description"`
	Body               string         `json:"body"`
	Labels             []string       `json:"labels"`
	Status             string         `json:"status"`
	AssignedTo         string         `json:"assigned_to"`
	Private            *bool          `json:"private"`
	DiscussionDisabled *bool          `json:"discussion_disabled"`
	CustomFields       map[string]any `json:"custom_fields"`
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

type actionsApplyResult struct {
	OK               bool                `json:"ok"`
	Confirmed        bool                `json:"confirmed"`
	Executed         bool                `json:"executed"`
	ValidatedActions []validatedAction   `json:"validated_actions"`
	AppliedActions   []appliedActionStep `json:"applied_actions,omitempty"`
}

type appliedActionStep struct {
	Index  int               `json:"index"`
	Type   string            `json:"type"`
	Target map[string]any    `json:"target,omitempty"`
	OK     bool              `json:"ok"`
	Issues []validationIssue `json:"issues,omitempty"`
}

func handleActions(ctx context.Context, client *api.Client, args []string) model.Envelope {
	if len(args) == 0 {
		return errorEnvelope("actions", proposal("actions", "dispatch_actions_command", nil, nil), "invalid_arguments", "missing actions subcommand\n\n"+actionsUsage())
	}

	switch args[0] {
	case "validate":
		return runActionsValidate(ctx, client, args[1:])
	case "apply":
		return runActionsApply(ctx, client, args[1:])
	default:
		command := "actions." + args[0]
		return errorEnvelope(command, proposal(command, actionForActions(args[0]), nil, nil), "not_implemented", fmt.Sprintf("command %q is not implemented yet\n\n%s", command, actionsUsage()))
	}
}

func runActionsApply(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseActionsApplyFlags(args)
	command := "actions.apply"
	prop := proposal(command, "apply_actions_file", nil, map[string]any{"file": config.ActionFile, "confirm": config.Confirm})
	if err != nil {
		return errorEnvelopeMode("dry_run", command, prop, "invalid_arguments", err.Error(), nil)
	}

	input, err := readActionsFile(config.ActionFile)
	if err != nil {
		return errorEnvelopeMode("dry_run", command, prop, "invalid_input", err.Error(), nil)
	}

	validated, err := validateIntentActions(ctx, client, input.Actions)
	if err != nil {
		apiErr := apiErrorEnvelope(command, prop, err)
		apiErr.Mode = "dry_run"
		return apiErr
	}

	result := actionsApplyResult{
		OK:               validated.OK,
		Confirmed:        config.Confirm,
		Executed:         false,
		ValidatedActions: validated.ValidatedActions,
	}
	if !validated.OK {
		return errorEnvelopeMode("dry_run", command, prop, "invalid_actions", "actions file contains invalid actions; run `sf actions validate` for details", result)
	}
	if !config.Confirm {
		return successEnvelopeMode("dry_run", command, prop, result)
	}
	if !client.HasToken() {
		return errorEnvelopeMode("dry_run", command, prop, "authentication_required", "confirmed apply requires a bearer token via `--token` or `SF_BEARER_TOKEN`", result)
	}

	if applied, ok := unsupportedApplyPlan(validated.ValidatedActions); !ok {
		result.OK = false
		result.AppliedActions = applied
		return errorEnvelopeMode("apply", command, prop, "unsupported_action_type", "confirmed apply currently supports `ticket_create`, `ticket_comment`, and `ticket_labels` actions only", result)
	}

	result.AppliedActions = make([]appliedActionStep, 0, len(validated.ValidatedActions))
	result.Executed = true
	for i, action := range input.Actions {
		applied := appliedActionStep{
			Index:  validated.ValidatedActions[i].Index,
			Type:   validated.ValidatedActions[i].Type,
			Target: validated.ValidatedActions[i].Target,
			OK:     true,
		}

		if err := applyConfirmedAction(ctx, client, validated.ValidatedActions[i], action); err != nil {
			issue := applyErrorIssue(err)
			applied.OK = false
			applied.Issues = []validationIssue{issue}
			result.OK = false
			result.AppliedActions = append(result.AppliedActions, applied)
			return errorEnvelopeMode("apply", command, prop, issue.Code, issue.Message, result)
		}

		result.AppliedActions = append(result.AppliedActions, applied)
	}

	result.OK = true
	return successEnvelopeMode("apply", command, prop, result)
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

func parseActionsApplyFlags(args []string) (actionsApplyConfig, error) {
	fs := flag.NewFlagSet("actions apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := actionsApplyConfig{}
	fs.BoolVar(&config.Confirm, "confirm", false, "Execute validated actions instead of stopping at dry-run")

	if err := fs.Parse(args); err != nil {
		return actionsApplyConfig{}, normalizeFlagError(err)
	}
	if fs.NArg() != 1 {
		return actionsApplyConfig{}, fmt.Errorf("missing required actions file\n\n%s", actionsApplyUsage())
	}

	config.ActionFile = strings.TrimSpace(fs.Arg(0))
	return config, nil
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
		Index:  index,
		Type:   action.Type,
		Target: actionTarget(action),
		OK:     true,
	}
	if trimmedType == actionTypeTicketCreate {
		validated.Action = normalizeTicketCreateAction(action)
		validated.CanonicalIdentifiers = ticketCreateCanonicalIdentifiers(trimmedProject, trimmedTracker)
	}
	if trimmedType == actionTypeTicketComment {
		validated.Action = normalizeTicketCommentAction(action)
		validated.CanonicalIdentifiers = ticketCommentCanonicalIdentifiers(trimmedProject, trimmedTracker, action.Ticket, "")
	}
	if trimmedType == actionTypeTicketLabels {
		validated.Action = normalizeTicketLabelsAction(action)
		validated.CanonicalIdentifiers = ticketLabelsCanonicalIdentifiers(trimmedProject, trimmedTracker, action.Ticket)
	}

	if trimmedType != actionTypeTicketCreate && trimmedType != actionTypeTicketComment && trimmedType != actionTypeTicketLabels {
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
	if requiresExistingTicket(trimmedType) && action.Ticket <= 0 {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "invalid_ticket", Field: "ticket", Message: "ticket must be > 0"})
	}

	switch trimmedType {
	case actionTypeTicketCreate:
		if strings.TrimSpace(action.Summary) == "" {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "missing_summary", Field: "summary", Message: "summary is required"})
		}
		if action.Ticket != 0 {
			validated.OK = false
			validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "unsupported_ticket_target", Field: "ticket", Message: "ticket must be omitted for ticket_create"})
		}
		appendTicketCreateFieldValidationIssues(&validated, action)
		appendLabelsValidationIssues(&validated, action.Labels)
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
		appendLabelsValidationIssues(&validated, action.Labels)
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
	if trimmedType == actionTypeTicketCreate {
		validated.CanonicalIdentifiers = ticketCreateCanonicalIdentifiers(project.Shortname, trimmedTracker)
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

func normalizeTicketCreateAction(action intentAction) map[string]any {
	inputs := map[string]any{
		"summary": strings.TrimSpace(action.Summary),
		"status":  normalizedTicketCreateStatus(action.Status),
	}
	if action.Private != nil {
		inputs["private"] = *action.Private
	}
	if action.Description != "" {
		inputs["description"] = action.Description
	}
	if len(action.CustomFields) != 0 {
		customFields := make(map[string]any, len(action.CustomFields))
		for key, value := range action.CustomFields {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			customFields[trimmedKey] = value
		}
		if len(customFields) != 0 {
			inputs["custom_fields"] = customFields
		}
	}
	if len(action.Labels) != 0 {
		labels := make([]string, 0, len(action.Labels))
		for _, label := range action.Labels {
			labels = append(labels, strings.TrimSpace(label))
		}
		inputs["labels"] = labels
	}

	return map[string]any{
		"type": actionTypeTicketCreate,
		"target": map[string]any{
			"project": strings.TrimSpace(action.Project),
			"tracker": strings.TrimSpace(action.Tracker),
		},
		"inputs": inputs,
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

func ticketCreateCanonicalIdentifiers(project string, tracker string) map[string]any {
	return ticketCommentCanonicalIdentifiers(project, tracker, 0, "")
}

func requiresExistingTicket(actionType string) bool {
	return actionType == actionTypeTicketComment || actionType == actionTypeTicketLabels
}

func actionTarget(action intentAction) map[string]any {
	target := map[string]any{
		"project": action.Project,
		"tracker": action.Tracker,
	}
	if requiresExistingTicket(strings.TrimSpace(action.Type)) || action.Ticket != 0 {
		target["ticket"] = action.Ticket
	}
	return target
}

func unsupportedApplyPlan(validated []validatedAction) ([]appliedActionStep, bool) {
	hasUnsupported := false
	for _, action := range validated {
		trimmedType := strings.TrimSpace(action.Type)
		if trimmedType != actionTypeTicketCreate && trimmedType != actionTypeTicketComment && trimmedType != actionTypeTicketLabels {
			hasUnsupported = true
			break
		}
	}
	if !hasUnsupported {
		return nil, true
	}

	applied := make([]appliedActionStep, 0, len(validated))
	for _, action := range validated {
		issue := validationIssue{
			Severity: "error",
			Code:     "apply_aborted",
			Message:  "action was not executed because the file includes unsupported confirmed apply action types",
		}
		trimmedType := strings.TrimSpace(action.Type)
		if trimmedType != actionTypeTicketCreate && trimmedType != actionTypeTicketComment && trimmedType != actionTypeTicketLabels {
			issue = validationIssue{
				Severity: "error",
				Code:     "unsupported_action_type",
				Field:    "type",
				Message:  fmt.Sprintf("action type %q is not enabled for apply yet", action.Type),
			}
		}
		applied = append(applied, appliedActionStep{
			Index:  action.Index,
			Type:   action.Type,
			Target: action.Target,
			OK:     false,
			Issues: []validationIssue{issue},
		})
	}
	return applied, false
}

func applyConfirmedAction(ctx context.Context, client *api.Client, validated validatedAction, action intentAction) error {
	switch strings.TrimSpace(validated.Type) {
	case actionTypeTicketComment:
		return applyTicketCommentAction(ctx, client, validated, action)
	case actionTypeTicketLabels:
		return applyTicketLabelsAction(ctx, client, validated, action)
	case actionTypeTicketCreate:
		return applyTicketCreateAction(ctx, client, validated)
	default:
		return fmt.Errorf("unsupported confirmed action type %q", validated.Type)
	}
}

func applyTicketCreateAction(ctx context.Context, client *api.Client, validated validatedAction) error {
	canonical := validated.CanonicalIdentifiers
	project, _ := canonical["project"].(string)
	tracker, _ := canonical["tracker"].(string)

	inputs := actionInputs(validated)
	summary, _ := inputs["summary"].(string)
	description, _ := inputs["description"].(string)

	return client.CreateTicket(ctx, api.CreateTicketParams{
		Project:      project,
		Tracker:      tracker,
		Status:       actionInputString(inputs, "status"),
		Private:      actionInputBoolPointer(inputs, "private"),
		Summary:      summary,
		Description:  description,
		CustomFields: actionInputMap(inputs, "custom_fields"),
		Labels:       actionInputStrings(inputs, "labels"),
	})
}

func applyTicketCommentAction(ctx context.Context, client *api.Client, validated validatedAction, action intentAction) error {
	canonical := validated.CanonicalIdentifiers
	project, _ := canonical["project"].(string)
	tracker, _ := canonical["tracker"].(string)
	threadID, _ := canonical["discussion_thread_id"].(string)
	inputs := actionInputs(validated)
	text, _ := inputs["body"].(string)
	if text == "" {
		text = action.Body
	}

	return client.CreateDiscussionPost(ctx, api.CreateDiscussionPostParams{
		Project:  project,
		Tracker:  tracker,
		ThreadID: threadID,
		Text:     text,
	})
}

func applyTicketLabelsAction(ctx context.Context, client *api.Client, validated validatedAction, action intentAction) error {
	canonical := validated.CanonicalIdentifiers
	project, _ := canonical["project"].(string)
	tracker, _ := canonical["tracker"].(string)
	ticketNum, _ := canonical["ticket_num"].(int)
	if ticketNum == 0 {
		if rawTicketNum, ok := canonical["ticket_num"].(float64); ok {
			ticketNum = int(rawTicketNum)
		}
	}

	labels := actionInputStrings(actionInputs(validated), "labels")
	if len(labels) == 0 {
		labels = make([]string, 0, len(action.Labels))
		for _, label := range action.Labels {
			labels = append(labels, strings.TrimSpace(label))
		}
	}

	return client.SaveTicketLabels(ctx, api.SaveTicketLabelsParams{
		Project: project,
		Tracker: tracker,
		Ticket:  ticketNum,
		Labels:  labels,
	})
}

func actionInputs(validated validatedAction) map[string]any {
	inputs, _ := validated.Action["inputs"].(map[string]any)
	return inputs
}

func actionInputString(inputs map[string]any, key string) string {
	value, _ := inputs[key].(string)
	return value
}

func actionInputBoolPointer(inputs map[string]any, key string) *bool {
	value, ok := inputs[key].(bool)
	if !ok {
		return nil
	}
	result := value
	return &result
}

func actionInputMap(inputs map[string]any, key string) map[string]any {
	value, _ := inputs[key].(map[string]any)
	return value
}

func actionInputStrings(inputs map[string]any, key string) []string {
	switch raw := inputs[key].(type) {
	case []string:
		values := make([]string, 0, len(raw))
		for _, value := range raw {
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	case []any:
		values := make([]string, 0, len(raw))
		for _, item := range raw {
			value, _ := item.(string)
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	default:
		return nil
	}
}

func applyErrorIssue(err error) validationIssue {
	code := "request_error"
	if _, ok := err.(*api.APIError); ok {
		code = "api_error"
	}

	return validationIssue{
		Severity: "error",
		Code:     code,
		Message:  err.Error(),
	}
}

func appendLabelsValidationIssues(validated *validatedAction, labels []string) {
	for i, label := range labels {
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

func appendTicketCreateFieldValidationIssues(validated *validatedAction, action intentAction) {
	if strings.TrimSpace(action.AssignedTo) != "" {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "unsupported_ticket_create_field", Field: "assigned_to", Message: "assigned_to is not supported for ticket_create"})
	}
	if action.DiscussionDisabled != nil {
		validated.OK = false
		validated.Issues = append(validated.Issues, validationIssue{Severity: "error", Code: "unsupported_ticket_create_field", Field: "discussion_disabled", Message: "discussion_disabled is not supported for ticket_create"})
	}
}

func normalizedTicketCreateStatus(status string) string {
	trimmed := strings.TrimSpace(status)
	if trimmed == "" {
		return "open"
	}
	return trimmed
}

func projectHasTracker(tools []api.ProjectTool, tracker string) bool {
	for _, tool := range tools {
		if tool.MountPoint == tracker {
			return true
		}
	}
	return false
}
