package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"sf-cli/internal/api"
	"sf-cli/internal/model"
)

type projectToolsConfig struct {
	Project string
}

func runProjectTools(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseProjectToolsFlags(args)
	command := "project.tools"
	prop := proposal(command, "list_project_tools", map[string]any{"project": config.Project}, nil)
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetProject(ctx, config.Project)
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}

	return successEnvelope(command, prop, result)
}

func parseProjectToolsFlags(args []string) (projectToolsConfig, error) {
	fs := flag.NewFlagSet("project tools", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := projectToolsConfig{}
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")

	if err := fs.Parse(args); err != nil {
		return projectToolsConfig{}, normalizeFlagError(err)
	}
	if strings.TrimSpace(config.Project) == "" {
		return projectToolsConfig{}, fmt.Errorf("missing required --project")
	}

	config.Project = strings.TrimSpace(config.Project)
	return config, nil
}
