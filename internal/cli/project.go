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
	Fields  []string
}

var projectToolProjectionFields = []string{"name", "mount_point", "mount_label", "url", "api_url", "clone_url_https_anon", "clone_url_ro"}

var projectToolFieldProjectors = map[string]func(api.ProjectTool) any{
	"api_url":              func(tool api.ProjectTool) any { return tool.APIURL },
	"clone_url_https_anon": func(tool api.ProjectTool) any { return tool.CloneURL },
	"clone_url_ro":         func(tool api.ProjectTool) any { return tool.CloneURLRO },
	"mount_label":          func(tool api.ProjectTool) any { return tool.MountLabel },
	"mount_point":          func(tool api.ProjectTool) any { return tool.MountPoint },
	"name":                 func(tool api.ProjectTool) any { return tool.Name },
	"url":                  func(tool api.ProjectTool) any { return tool.URL },
}

func runProjectTools(ctx context.Context, client *api.Client, args []string) model.Envelope {
	config, err := parseProjectToolsFlags(args)
	command := "project.tools"
	prop := proposal(command, "list_project_tools", map[string]any{"project": config.Project}, projectToolsInputs(config))
	if err != nil {
		return errorEnvelope(command, prop, "invalid_arguments", err.Error())
	}

	result, err := client.GetProject(ctx, config.Project)
	if err != nil {
		return apiErrorEnvelope(command, prop, err)
	}
	if len(config.Fields) != 0 {
		return successEnvelope(command, prop, projectToolsResult(result.Tools, config.Fields))
	}

	return successEnvelope(command, prop, result)
}

func parseProjectToolsFlags(args []string) (projectToolsConfig, error) {
	fs := flag.NewFlagSet("project tools", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	config := projectToolsConfig{}
	var rawFields string
	fs.StringVar(&config.Project, "project", "", "SourceForge project shortname")
	fs.StringVar(&rawFields, "fields", "", "Comma-separated projected tool fields")

	if err := fs.Parse(args); err != nil {
		return projectToolsConfig{}, normalizeFlagError(err)
	}
	if strings.TrimSpace(config.Project) == "" {
		return projectToolsConfig{}, fmt.Errorf("missing required --project")
	}

	config.Project = strings.TrimSpace(config.Project)
	fields, err := parseFields(rawFields, hasFlag(args, "fields"), projectToolProjectionFields, "project tools")
	if err != nil {
		return projectToolsConfig{}, err
	}
	config.Fields = fields
	return config, nil
}

func projectToolsInputs(config projectToolsConfig) map[string]any {
	if len(config.Fields) == 0 {
		return nil
	}
	return map[string]any{"fields": config.Fields}
}

func projectToolsResult(tools []api.ProjectTool, fields []string) map[string]any {
	projectedTools := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		projected := make(map[string]any, len(fields))
		for _, field := range fields {
			projected[field] = projectToolFieldProjectors[field](tool)
		}
		projectedTools = append(projectedTools, projected)
	}
	return map[string]any{"tools": projectedTools}
}
