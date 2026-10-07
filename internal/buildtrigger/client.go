// Package buildtrigger dispatches the build of a tenant's app to Storeapp's GitHub Actions workflow.
package buildtrigger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/env"
	"github.com/google/uuid"
)

const (
	// GitHubAPIURL is the endpoint every client uses outside of tests.
	GitHubAPIURL = "https://api.github.com"

	requestTimeout   = 10 * time.Second
	maxResponseBytes = 64 * 1024
	// maxMessageLength bounds what of GitHub's answer ends up in a record the operator reads.
	maxMessageLength = 200

	// TenantConfigIDInput is the name of the workflow_dispatch input that carries the TenantConfig id. The workflow
	// of Storeapp (build-tenant-apk.yml) declares it as build_id, and GitHub refuses a dispatch with an input the
	// workflow does not declare. The value is still the id of the TenantConfig record.
	TenantConfigIDInput = "build_id"
)

// ErrNotConfigured means no GitHub token is configured, so no build can be dispatched.
var ErrNotConfigured = errors.New("tenant build dispatch is not configured")

type Client struct {
	config   env.StoreappBuildConfig
	endpoint string
	client   *http.Client
}

// NewClient creates a client dispatching to the given workflow. An empty endpoint means GitHubAPIURL.
func NewClient(config env.StoreappBuildConfig, endpoint string) *Client {
	if endpoint == "" {
		endpoint = GitHubAPIURL
	}
	return &Client{
		config:   config,
		endpoint: strings.TrimSuffix(endpoint, "/"),
		client:   &http.Client{Timeout: requestTimeout},
	}
}

type dispatchRequest struct {
	Ref    string            `json:"ref"`
	Inputs map[string]string `json:"inputs"`
}

type errorResponse struct {
	Message string `json:"message"`
}

// Dispatch asks GitHub to run the workflow for one tenant. GitHub answers 204 with no run id, so success means the
// request was accepted, not that a build exists yet or that it will succeed.
func (c *Client) Dispatch(ctx context.Context, tenantConfigID uuid.UUID) error {
	body, err := json.Marshal(dispatchRequest{
		Ref:    c.config.Ref,
		Inputs: map[string]string{TenantConfigIDInput: tenantConfigID.String()},
	})
	if err != nil {
		return fmt.Errorf("could not encode dispatch request: %w", err)
	}

	// The repository is validated as owner/name at startup. The workflow is escaped because it is a path segment.
	endpoint := fmt.Sprintf("%v/repos/%v/actions/workflows/%v/dispatches",
		c.endpoint, c.config.Repository, url.PathEscape(c.config.Workflow))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not create dispatch request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		// The error of the HTTP client carries the URL, never a header, so the token cannot leak through it.
		return fmt.Errorf("dispatch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return fmt.Errorf("GitHub refused the dispatch with status %v%v", resp.StatusCode, githubMessage(resp.Body))
}

// githubMessage renders the message GitHub gave for a refusal, e.g. that the workflow has no workflow_dispatch
// trigger or does not declare the input, which is what an operator needs to read.
func githubMessage(body io.Reader) string {
	var parsed errorResponse
	if err := json.NewDecoder(io.LimitReader(body, maxResponseBytes)).Decode(&parsed); err != nil || parsed.Message == "" {
		return ""
	}
	// Cut on a rune boundary: the message is stored in a text column, which rejects invalid UTF-8.
	message := []rune(parsed.Message)
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength]
	}
	return ": " + string(message)
}
