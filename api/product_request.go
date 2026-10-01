package api

import (
	"bytes"
	"image/png"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/distr-sh/distr/internal/devicefingerprint"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

const (
	maxProductRequestEmailLength   = 254
	maxProductRequestAppNameLength = 100

	// MaxProductRequestBodyBytes bounds the whole request body, which carries the icon as base64. It is a little over
	// the encoded size of an icon of MaxProductRequestIconBytes.
	MaxProductRequestBodyBytes = 400 * 1024
	MaxProductRequestIconBytes = 256 * 1024
	// The icon has to be square, and neither so small that the launcher scales it up nor large enough to be a
	// decompression risk.
	minProductRequestIconSize = 96
	maxProductRequestIconSize = 1024

	ProductRequestIconContentType = "image/png"
)

var productRequestThemeColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type CreateProductRequestRequest struct {
	ContactEmail string `json:"contactEmail"`
	AppName      string `json:"appName"`
	// ProductServiceID is the storefront entry that is requested.
	ProductServiceID uuid.UUID `json:"productServiceId"`
	// ThemeColor is a hex color in the form #rrggbb.
	ThemeColor string `json:"themeColor"`
	// Icon is optional.
	Icon *ProductRequestIcon `json:"icon,omitempty"`
	// TurnstileToken is required when the instance has Turnstile configured.
	TurnstileToken string `json:"turnstileToken,omitempty"`
	// DeviceFingerprint is the lower case hex SHA-256 of the device signals the browser computed. It is optional: a
	// request without one is accepted and is never free. It is only used to give each device one free product.
	DeviceFingerprint string `json:"deviceFingerprint,omitempty"`
}

// ProductRequestIcon is an uploaded icon. Data is base64 in JSON.
type ProductRequestIcon struct {
	ContentType string `json:"contentType"`
	Data        []byte `json:"data" trim:"-"`
}

func (i ProductRequestIcon) Validate() error {
	if i.ContentType != ProductRequestIconContentType {
		return validation.NewValidationFailedError("the icon must be a PNG image")
	}
	if len(i.Data) == 0 {
		return validation.NewValidationFailedError("the icon is empty")
	}
	if len(i.Data) > MaxProductRequestIconBytes {
		return validation.NewValidationFailedError("the icon is too large, the limit is 256 KB")
	}
	// The declared content type is the requester's word. Reading the header is what proves the bytes are a PNG
	// and gives the dimensions without decoding the pixels.
	config, err := png.DecodeConfig(bytes.NewReader(i.Data))
	if err != nil {
		return validation.NewValidationFailedError("the icon is not a valid PNG image")
	}
	if config.Width != config.Height {
		return validation.NewValidationFailedError("the icon must be square")
	}
	if config.Width < minProductRequestIconSize || config.Width > maxProductRequestIconSize {
		return validation.NewValidationFailedError("the icon must be between 96 and 1024 pixels wide")
	}
	return nil
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
	if r.ProductServiceID == uuid.Nil {
		return validation.NewValidationFailedError("choose a product")
	}
	if !productRequestThemeColorPattern.MatchString(r.ThemeColor) {
		return validation.NewValidationFailedError("theme color must be a hex color such as #1a2b3c")
	}
	if r.Icon != nil {
		if err := r.Icon.Validate(); err != nil {
			return err
		}
	}
	if r.DeviceFingerprint != "" && !devicefingerprint.ValidClientHash(r.DeviceFingerprint) {
		return validation.NewValidationFailedError("the device check is malformed, please reload the page and try again")
	}
	return nil
}

type ProductRequest struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	// TenantID is the permanent identifier of the tenant record created with the request.
	TenantID string `json:"tenantId"`
	// BuildStatus is awaiting_gate until the free/paid check has cleared the request.
	BuildStatus types.TenantBuildStatus `json:"buildStatus"`
	// Gate is free when the request used the free product of its device, and payment_required when it has to be paid
	// for before anything is built.
	Gate types.ProductRequestGate `json:"gate"`
	// PaymentURL is the B-Pay page that takes the payment. It is set when the request has to be paid for, the product
	// has a price and B-Pay accepted the payment, and absent otherwise (ask for it again with the payment endpoint).
	PaymentURL *string `json:"paymentUrl,omitempty"`
}

// ProductRequestPayment is the answer of the payment endpoint of a request.
type ProductRequestPayment struct {
	PaymentURL string `json:"paymentUrl"`
}
