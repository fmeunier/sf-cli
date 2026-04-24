package api

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type TrackerSchemaResponse struct {
	Project    string         `json:"project"`
	Tracker    string         `json:"tracker"`
	Options    map[string]any `json:"options,omitempty"`
	Milestones []Milestone    `json:"milestones,omitempty"`
	SavedBins  []SavedBin     `json:"saved_bins,omitempty"`
	Fields     []TrackerField `json:"fields,omitempty"`
	Warnings   []string       `json:"warnings,omitempty"`
}

type SavedBin struct {
	ID      string `json:"id,omitempty"`
	Summary string `json:"summary,omitempty"`
	Terms   string `json:"terms,omitempty"`
	Sort    string `json:"sort,omitempty"`
}

type TrackerField struct {
	Name       string                  `json:"name"`
	Values     []TrackerFieldValue     `json:"values,omitempty"`
	Validation *TrackerFieldValidation `json:"validation,omitempty"`
}

type TrackerFieldValue struct {
	Value any  `json:"value"`
	Count *int `json:"count,omitempty"`
}

type TrackerFieldValidation struct {
	Type          string              `json:"type"`
	Required      *bool               `json:"required,omitempty"`
	Default       any                 `json:"default,omitempty"`
	AllowedValues []TrackerFieldValue `json:"allowed_values,omitempty"`
}

func (c *Client) GetTrackerSchema(ctx context.Context, project string, tracker string) (TrackerSchemaResponse, error) {
	result := TrackerSchemaResponse{
		Project: project,
		Tracker: tracker,
	}

	var listPayload map[string]any
	listErr := c.GetJSON(ctx, trackerPath(project, tracker), nil, &listPayload)
	if listErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("tracker metadata unavailable: %s", listErr.Error()))
	} else {
		result.Options = trackerOptions(listPayload)
		result.Milestones = trackerMilestones(listPayload)
		result.SavedBins = trackerSavedBins(listPayload)
	}

	var searchPayload map[string]any
	searchQuery := url.Values{}
	searchQuery.Set("q", "*:*")
	searchQuery.Set("limit", "1")
	searchErr := c.GetJSON(ctx, trackerPath(project, tracker)+"/search", searchQuery, &searchPayload)
	if searchErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("tracker filter metadata unavailable: %s", searchErr.Error()))
	} else {
		result.Fields = trackerFields(searchPayload, result.Milestones)
	}

	if listErr != nil && searchErr != nil {
		return TrackerSchemaResponse{}, fmt.Errorf("tracker schema unavailable: %w; %v", listErr, searchErr)
	}

	return result, nil
}

func trackerOptions(payload map[string]any) map[string]any {
	trackerConfig, ok := payload["tracker_config"].(map[string]any)
	if !ok {
		return nil
	}
	options, ok := trackerConfig["options"].(map[string]any)
	if !ok || len(options) == 0 {
		return nil
	}
	return options
}

func trackerMilestones(payload map[string]any) []Milestone {
	rawMilestones, ok := payload["milestones"].([]any)
	if !ok {
		return nil
	}

	milestones := make([]Milestone, 0, len(rawMilestones))
	for _, rawMilestone := range rawMilestones {
		milestoneMap, ok := rawMilestone.(map[string]any)
		if !ok {
			continue
		}
		milestones = append(milestones, Milestone{
			Name:        stringValue(milestoneMap["name"]),
			Description: stringValue(milestoneMap["description"]),
			DueDate:     stringValue(milestoneMap["due_date"]),
			Default:     milestoneMap["default"],
			Complete:    boolValue(milestoneMap["complete"]),
			Closed:      intValue(milestoneMap["closed"]),
			Total:       intValue(milestoneMap["total"]),
		})
	}

	if len(milestones) == 0 {
		return nil
	}
	return milestones
}

func trackerSavedBins(payload map[string]any) []SavedBin {
	rawSavedBins, ok := payload["saved_bins"].([]any)
	if !ok {
		return nil
	}

	savedBins := make([]SavedBin, 0, len(rawSavedBins))
	for _, rawSavedBin := range rawSavedBins {
		savedBinMap, ok := rawSavedBin.(map[string]any)
		if !ok {
			continue
		}
		savedBins = append(savedBins, SavedBin{
			ID:      stringValue(savedBinMap["_id"]),
			Summary: stringValue(savedBinMap["summary"]),
			Terms:   stringValue(savedBinMap["terms"]),
			Sort:    stringValue(savedBinMap["sort"]),
		})
	}

	if len(savedBins) == 0 {
		return nil
	}
	return savedBins
}

func trackerFields(payload map[string]any, milestones []Milestone) []TrackerField {
	rawFilterChoices, ok := payload["filter_choices"].(map[string]any)
	if !ok || len(rawFilterChoices) == 0 {
		return nil
	}

	names := make([]string, 0, len(rawFilterChoices))
	for name := range rawFilterChoices {
		names = append(names, name)
	}
	sort.Strings(names)

	fields := make([]TrackerField, 0, len(names))
	for _, name := range names {
		values := trackerFieldValues(rawFilterChoices[name])
		fields = append(fields, TrackerField{
			Name:       name,
			Values:     values,
			Validation: trackerFieldValidation(name, values, milestones),
		})
	}

	if len(fields) == 0 {
		return nil
	}
	return fields
}

func trackerFieldValidation(name string, values []TrackerFieldValue, milestones []Milestone) *TrackerFieldValidation {
	validationType := "unknown"
	if allTrackerFieldValuesComparable(values) {
		validationType = "choice"
	}

	validation := &TrackerFieldValidation{Type: validationType}
	if len(values) > 0 && validationType == "choice" {
		validation.AllowedValues = append([]TrackerFieldValue(nil), values...)
	}

	if name == "_milestone" {
		if defaultValue, ok := defaultMilestoneValue(milestones); ok {
			validation.Default = defaultValue
		}
	}

	if len(validation.AllowedValues) == 0 && validation.Default == nil && validation.Required == nil && validation.Type == "unknown" {
		return validation
	}

	return validation
}

func allTrackerFieldValuesComparable(values []TrackerFieldValue) bool {
	if len(values) == 0 {
		return false
	}

	for _, value := range values {
		switch value.Value.(type) {
		case string, bool, int, int32, int64, float64:
			continue
		default:
			return false
		}
	}

	return true
}

func defaultMilestoneValue(milestones []Milestone) (any, bool) {
	for _, milestone := range milestones {
		if truthyValue(milestone.Default) {
			return milestone.Name, true
		}
	}
	return nil, false
}

func truthyValue(raw any) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(value))
		return trimmed == "1" || trimmed == "true" || trimmed == "yes" || trimmed == "on"
	default:
		return false
	}
}

func trackerFieldValues(raw any) []TrackerFieldValue {
	rawValues, ok := raw.([]any)
	if !ok {
		if raw == nil {
			return nil
		}
		return []TrackerFieldValue{{Value: raw}}
	}

	values := make([]TrackerFieldValue, 0, len(rawValues))
	for _, rawValue := range rawValues {
		switch typed := rawValue.(type) {
		case []any:
			if len(typed) == 0 {
				continue
			}
			entry := TrackerFieldValue{Value: typed[0]}
			if len(typed) > 1 {
				count := intValue(typed[1])
				entry.Count = &count
			}
			values = append(values, entry)
		default:
			values = append(values, TrackerFieldValue{Value: typed})
		}
	}

	if len(values) == 0 {
		return nil
	}
	return values
}

func stringValue(raw any) string {
	if value, ok := raw.(string); ok {
		return value
	}
	return ""
}

func boolValue(raw any) bool {
	if value, ok := raw.(bool); ok {
		return value
	}
	return false
}

func intValue(raw any) int {
	switch value := raw.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return 0
}
