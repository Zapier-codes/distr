// Package novu triggers a workflow of a Novu instance, which is how distr hands a mail off without sending it itself.
package novu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/env"
)

const (
	requestTimeout   = 10 * time.Second
	maxResponseBytes = 64 * 1024
	maxMessageLength = 200
)

// ErrNotConfigured means NOVU_API_KEY is not set, so no mail can be handed off.
var ErrNotConfigured = errors.New("the mail hand-off to Novu is not configured")

type Client struct {
	config env.NovuConfig
	client *http.Client
}

func NewClient(config env.NovuConfig) *Client {
	return &Client{config: config, client: &http.Client{Timeout: requestTimeout}}
}

// FromEnv returns a client for the configured instance, or ErrNotConfigured.
func FromEnv() (*Client, error) {
	config := env.Novu()
	if config == nil {
		return nil, ErrNotConfigured
	}
	return NewClient(*config), nil
}

type triggerRequest struct {
	Name    string            `json:"name"`
	To      triggerSubscriber `json:"to"`
	Payload map[string]string `json:"payload"`
}

type triggerSubscriber struct {
	SubscriberID string `json:"subscriberId"`
	Email        string `json:"email"`
}

type errorResponse struct {
	Message any `json:"message"`
}

// Trigger starts a workflow for one subscriber. Novu creates the subscriber from the email when it does not know the
// id yet. Success means Novu accepted the trigger, not that the mail was delivered.
func (c *Client) Trigger(
	ctx context.Context, workflowID, subscriberID, email string, payload map[string]string,
) error {
	body, err := json.Marshal(triggerRequest{
		Name:    workflowID,
		To:      triggerSubscriber{SubscriberID: subscriberID, Email: email},
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("could not encode Novu trigger: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.config.APIURL+"/v1/events/trigger", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not create Novu trigger: %w", err)
	}
	req.Header.Set("Authorization", "ApiKey "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		// The error of the HTTP client carries the URL, never a header, so the key cannot leak through it.
		return fmt.Errorf("Novu trigger failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("Novu refused the trigger with status %v%v", resp.StatusCode, novuMessage(resp.Body))
}

func novuMessage(body io.Reader) string {
	var parsed errorResponse
	if err := json.NewDecoder(io.LimitReader(body, maxResponseBytes)).Decode(&parsed); err != nil || parsed.Message == nil {
		return ""
	}
	// Novu answers a string, or a list of strings for a validation failure.
	text := strings.TrimSpace(fmt.Sprint(parsed.Message))
	if text == "" {
		return ""
	}
	message := []rune(text)
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength]
	}
	return ": " + string(message)
}
