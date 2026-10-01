package api

import (
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

const (
	maxProductRequestEmailLength   = 254
	maxProductRequestAppNameLength = 100
)

var productRequestThemeColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type CreateProductRequestRequest struct {
	ContactEmail string `json:"contactEmail"`
	AppName      string `json:"appName"`
	// ThemeColor is a hex color in the form #rrggbb.
	ThemeColor string `json:"themeColor"`
	// TurnstileToken is required when the instance has Turnstile configured.
	TurnstileToken string `json:"turnstileToken,omitempty"`
}

func (r *CreateProductRequestRequest) Validate() error {
	if r.ContactEmail == "" {
		return validation.NewValidationFailedError("email is empty")
	} else if len(r.ContactEmail) > maxProductRequestEmailLength {
		return validation.NewValidationFailedError("email is too long")
	} else if err := validation.ValidateEmail(r.ContactEmail); err != nil {
		return err
	}
	if r.AppName == "" {
		return validation.NewValidationFailedError("app name is empty")
	} else if utf8.RuneCountInString(r.AppName) > maxProductRequestAppNameLength {
		return validation.NewValidationFailedError("app name is too long")
	}
	if !productRequestThemeColorPattern.MatchString(r.ThemeColor) {
		return validation.NewValidationFailedError("theme color must be a hex color such as #1a2b3c")
	}
	return nil
}

type ProductRequest struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}
