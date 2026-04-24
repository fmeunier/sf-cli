package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const DefaultBaseURL = "https://sourceforge.net/rest"

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
}

type Options struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("api request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("api request failed with status %d: %s", e.StatusCode, e.Message)
}

func NewClient(opts Options) (*Client, error) {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    parsedBaseURL,
		httpClient: httpClient,
		token:      strings.TrimSpace(opts.Token),
	}, nil
}

func (c *Client) NewRequest(ctx context.Context, method string, endpointPath string, query url.Values) (*http.Request, error) {
	endpointURL := *c.baseURL
	endpointURL.Path = path.Join(c.baseURL.Path, endpointPath)
	if len(query) > 0 {
		endpointURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpointURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return req, nil
}

func (c *Client) GetJSON(ctx context.Context, endpointPath string, query url.Values, out any) error {
	req, err := c.NewRequest(ctx, http.MethodGet, endpointPath, query)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(resp)
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func decodeAPIError(resp *http.Response) error {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	_ = json.NewDecoder(resp.Body).Decode(&payload)

	message := strings.TrimSpace(payload.Error)
	if message == "" {
		message = strings.TrimSpace(payload.Message)
	}

	return &APIError{StatusCode: resp.StatusCode, Message: message}
}
