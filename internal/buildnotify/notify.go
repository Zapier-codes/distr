// Package buildnotify mails the requester that the build of their app is ready (leaf f.v). The mail is sent by a
// Novu workflow; this package decides when to trigger it and what link it carries.
package buildnotify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/novu"
	"github.com/google/uuid"
)

const (
	// DownloadPath is the public route that resolves a download token. The token is appended.
	DownloadPath = "/api/public/v1/build-downloads/"
	// DownloadValidity is how long the link in the mail works. The asset itself stays on the Release.
	DownloadValidity = 30 * 24 * time.Hour

	tokenBytes = 32
	// TokenLength is the length of an encoded token, which is how a handler tells a token from anything else.
	TokenLength = 43
)

// ErrNotConfigured means the hand-off to Novu is not set up, so nothing was sent and nothing was recorded.
var ErrNotConfigured = novu.ErrNotConfigured

// NewToken returns a random download token and its SHA-256. Only the hash is stored: the token is CSPRNG output,
// so a fast digest is enough, and the lookup is by the digest, never by comparing in Go.
func NewToken() (token string, hash []byte, err error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("could not generate download token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, HashToken(token), nil
}

func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// DownloadURL is the link the mail carries.
func DownloadURL(baseURL, token string) string {
	return baseURL + DownloadPath + token
}

// SendBuildReady mails the requester of a record whose build succeeded. It does nothing when the mail was already
// accepted by Novu, so it is safe to call again whenever the workflow repeats its report. A new token replaces the
// previous one on every attempt that reaches Novu, and the mail is marked sent only after Novu accepted it.
func SendBuildReady(ctx context.Context, tenantConfigID uuid.UUID, baseURL string) error {
	client, err := novu.FromEnv()
	if err != nil {
		return err
	}
	config := env.Novu()

	recipient, err := db.GetBuildReadyRecipient(ctx, tenantConfigID)
	if errors.Is(err, apierrors.ErrNotFound) {
		// A record that predates the storefront has no request, so there is nobody to mail.
		return nil
	} else if err != nil {
		return err
	}
	if recipient.EmailSentAt != nil {
		return nil
	}

	token, hash, err := NewToken()
	if err != nil {
		return err
	}
	if err := db.SetTenantBuildDownloadToken(ctx, tenantConfigID, hash, time.Now().Add(DownloadValidity)); err != nil {
		return err
	}
	if err := client.Trigger(ctx, config.BuildReadyWorkflowID, tenantConfigID.String(), recipient.ContactEmail,
		map[string]string{
			"appName":     recipient.AppName,
			"downloadUrl": DownloadURL(baseURL, token),
		},
	); err != nil {
		return err
	}
	return db.MarkTenantBuildEmailSent(ctx, tenantConfigID)
}
