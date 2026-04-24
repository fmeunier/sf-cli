package api

import (
	"context"
	"fmt"
)

type ProjectResponse struct {
	Shortname string        `json:"shortname"`
	Name      string        `json:"name"`
	URL       string        `json:"url"`
	Tools     []ProjectTool `json:"tools"`
}

type ProjectTool struct {
	Name       string `json:"name"`
	MountPoint string `json:"mount_point"`
	MountLabel string `json:"mount_label"`
	URL        string `json:"url"`
	APIURL     string `json:"api_url,omitempty"`
	CloneURL   string `json:"clone_url_https_anon,omitempty"`
	CloneURLRO string `json:"clone_url_ro,omitempty"`
}

func (c *Client) GetProject(ctx context.Context, project string) (ProjectResponse, error) {
	var out ProjectResponse
	if err := c.GetJSON(ctx, fmt.Sprintf("p/%s", project), nil, &out); err != nil {
		return ProjectResponse{}, err
	}
	return out, nil
}
