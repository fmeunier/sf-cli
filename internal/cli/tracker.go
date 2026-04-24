package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"sf-cli/internal/api"
	"sf-cli/internal/model"
)

type trackerSchemaConfig struct {
	Project string
	Tracker string
	Fields  []string
}

var trackerSchemaProjectionFields = []string{"project", "tracker", "options", "milestones", "saved_bins", "fields"}

func runTrackerSchema(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTrackerSchemaFlags(args)
	command := "tracker.schema"
	prop := proposal(command, "get_tracker_schema", map[string]any{"project": config.Project, "tracker": config.Tracker}, trackerSchemaInputs(config))
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetTrackerSchema(ctx, config.Project, config.Tracker)
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	warnings := append([]string{}, result.Warnings...)
	result.Warnings = nil
	if len(config.Fields) != 0 {
		return successEnvelope(command, prop, projectTrackerSchemaResult(result, config.Fields), warnings...)
	}

	return successEnvelope(command, prop, result, warnings...)
}

func parseTrackerSchemaFlags(args []string) (trackerSchemaConfig, error) {
	fs := flag.NewFlagSet("tracker schema", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := trackerSchemaConfig{}
	var rawFields string
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")
	fs.StringVar(&rawFields, "fields", "", "Comma-separated projected schema fields")

	if err := fs.Parse(args); err != nil {
		return trackerSchemaConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return trackerSchemaConfig{}, fmt.Errorf("%w", err)
	}
	fields, err := parseFields(rawFields, hasFlag(args, "fields"), trackerSchemaProjectionFields, "tracker schema")
	if err != nil {
		return trackerSchemaConfig{}, err
	}
	config.Fields = fields

	return config, nil
}

func trackerSchemaInputs(config trackerSchemaConfig) map[string]any {
	if len(config.Fields) == 0 {
		return nil
	}
	return map[string]any{"fields": config.Fields}
}

func projectTrackerSchemaResult(result api.TrackerSchemaResponse, fields []string) map[string]any {
	projected := make(map[string]any, len(fields))
	for _, field := range fields {
		switch field {
		case "project":
			projected[field] = result.Project
		case "tracker":
			projected[field] = result.Tracker
		case "options":
			projected[field] = result.Options
		case "milestones":
			projected[field] = result.Milestones
		case "saved_bins":
			projected[field] = result.SavedBins
		case "fields":
			projected[field] = result.Fields
		}
	}
	return projected
}
