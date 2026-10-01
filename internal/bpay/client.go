// Package bpay talks to B-Pay, Zapier-codes' own payment gateway (a Hyperswitch deployment). B-Pay hides the payment
// rails behind it, so Distr only ever creates a payment with a payment link and retrieves it again. No processor SDK
// is used and no processor is named anywhere in Distr.
package bpay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/env"
)

const (
	requestTimeout   = 10 * time.Second
	maxResponseBytes = 256 * 1024
	maxMessageLength = 200

	// SignatureHeader carries the hex HMAC-SHA512 of the raw webhook body, keyed with the webhook secret.
	SignatureHeader = "X-Webhook-Signature-512"

	// EventPaymentSucceeded is the event type of a payment that went through.
	EventPaymentSucceeded = "payment_succeeded"
	// StatusSucceeded is the status of a payment that went through.
	StatusSucceeded = "succeeded"
)

// ErrNotConfigured means B-Pay is not configured, so no payment can be created.
var ErrNotConfigured = errors.New("payments are not configured")

type Client struct {
	config env.BPayConfig
	client *http.Client
}

func NewClient(config env.BPayConfig) *Client {
	return &Client{config: config, client: &http.Client{Timeout: requestTimeout}}
}

// FromEnv returns a client for the configured B-Pay, or ErrNotConfigured.
func FromEnv() (*Client, error) {
	config := env.BPay()
	if config == nil {
		return nil, ErrNotConfigured
	}
	return NewClient(*config), nil
}

// Payment is what Distr needs of a B-Pay payment.
type Payment struct {
	ID          string
	Status      string
	AmountMinor int64
	Currency    string
	// Link is the page that takes the payment. Only set when the payment was created with a payment link.
	Link string
}

type createRequest struct {
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	PaymentLink bool              `json:"payment_link"`
	ReturnURL   string            `json:"return_url,omitempty"`
	Description string            `json:"description,omitempty"`
	ProfileID   *string           `json:"profile_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type paymentResponse struct {
	PaymentID   string `json:"payment_id"`
	Status      string `json:"status"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	PaymentLink *struct {
		Link string `json:"link"`
	} `json:"payment_link"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// CreatePayment creates a one-time payment with a payment link. Amount is in the minor unit of the currency.
// Metadata is kept by B-Pay with the payment and must hold nothing about the requester.
func (c *Client) CreatePayment(
	ctx context.Context, amountMinor int64, currency, returnURL, description string, metadata map[string]string,
) (*Payment, error) {
	body, err := json.Marshal(createRequest{
		Amount:      amountMinor,
		Currency:    currency,
		PaymentLink: true,
		ReturnURL:   returnURL,
		Description: description,
		ProfileID:   c.config.ProfileID,
		Metadata:    metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("could not encode B-Pay payment: %w", err)
	}
	payment, err := c.do(ctx, http.MethodPost, "/payments", body)
	if err != nil {
		return nil, err
	}
	if payment.Link == "" {
		return nil, errors.New("B-Pay created the payment without a payment link")
	}
	if u, err := url.Parse(payment.Link); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, errors.New("B-Pay answered a payment link that is not an http(s) URL")
	}
	return payment, nil
}

// GetPayment retrieves a payment. It is how a webhook is confirmed: the body of a webhook is only a hint, the
// payment as B-Pay itself reports it is what decides.
func (c *Client) GetPayment(ctx context.Context, paymentID string) (*Payment, error) {
	if paymentID == "" || strings.ContainsAny(paymentID, "/?#%\\ ") {
		return nil, errors.New("the payment id is malformed")
	}
	return c.do(ctx, http.MethodGet, "/payments/"+url.PathEscape(paymentID), nil)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*Payment, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.config.APIURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("could not create B-Pay request: %w", err)
	}
	req.Header.Set("api-key", c.config.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		// The error of the HTTP client carries the URL, never a header, so the key cannot leak through it.
		return nil, fmt.Errorf("B-Pay request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("B-Pay refused the request with status %v%v", resp.StatusCode, message(resp.Body))
	}
	var parsed paymentResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("could not decode the B-Pay answer: %w", err)
	}
	if parsed.PaymentID == "" {
		return nil, errors.New("the B-Pay answer has no payment id")
	}
	payment := &Payment{
		ID:          parsed.PaymentID,
		Status:      parsed.Status,
		AmountMinor: parsed.Amount,
		Currency:    strings.ToUpper(parsed.Currency),
	}
	if parsed.PaymentLink != nil {
		payment.Link = parsed.PaymentLink.Link
	}
	return payment, nil
}

func message(body io.Reader) string {
	var parsed errorResponse
	if err := json.NewDecoder(io.LimitReader(body, maxResponseBytes)).Decode(&parsed); err != nil {
		return ""
	}
	text := []rune(strings.TrimSpace(parsed.Error.Message))
	if len(text) == 0 {
		return ""
	}
	if len(text) > maxMessageLength {
		text = text[:maxMessageLength]
	}
	return ": " + string(text)
}

// VerifySignature reports whether signature (hex) is the HMAC-SHA512 of body under secret, comparing in constant
// time. A malformed signature is simply not valid.
func VerifySignature(secret string, body []byte, signature string) bool {
	given, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(given) != sha512.Size {
		return false
	}
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), given)
}

// WebhookEvent is the part of a B-Pay outgoing webhook that Distr reads.
type WebhookEvent struct {
	EventType string `json:"event_type"`
	Content   struct {
		Type   string `json:"type"`
		Object struct {
			PaymentID string `json:"payment_id"`
		} `json:"object"`
	} `json:"content"`
}

// ParseWebhookEvent reads a webhook body. The signature has to be verified before it is parsed.
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("could not decode the webhook: %w", err)
	}
	return &event, nil
}
