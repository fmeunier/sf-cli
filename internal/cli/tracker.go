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
}

func runTrackerSchema(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseTrackerSchemaFlags(args)
	command := "tracker.schema"
	prop := proposal(command, "get_tracker_schema", map[string]any{"project": config.Project, "tracker": config.Tracker}, nil)
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetTrackerSchema(ctx, config.Project, config.Tracker)
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

func parseTrackerSchemaFlags(args []string) (trackerSchemaConfig, error) {
	fs := flag.NewFlagSet("tracker schema", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := trackerSchemaConfig{}
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&config.Tracker, "tracker", "", "Tracker mount point")

	if err := fs.Parse(args); err != nil {
		return trackerSchemaConfig{}, normalizeFlagError(err)
	}
	if err := validateTrackerTarget(config.Project, config.Tracker); err != nil {
		return trackerSchemaConfig{}, fmt.Errorf("%w", err)
	}

	return config, nil
}
