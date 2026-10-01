package types

import (
	"time"

	"github.com/google/uuid"
)

// ProductRequest is a request for a product build that anyone can submit without an account. It has no
// reference to a user, an organization or a customer organization, by design.
type ProductRequest struct {
	ID           uuid.UUID `db:"id"`
	CreatedAt    time.Time `db:"created_at"`
	ContactEmail string    `db:"contact_email"`
	AppName      string    `db:"app_name"`
	// ThemeColor is a hex color in the form #rrggbb.
	ThemeColor string `db:"theme_color"`
	// TenantConfigID is the canonical tenant record created with the request. It is nil only for requests that
	// predate the storefront.
	TenantConfigID *uuid.UUID `db:"tenant_config_id"`
}

// ProductRequestGate is the outcome of the free/paid check a request goes through when it is submitted.
type ProductRequestGate string

const (
	// ProductRequestGateFree means the request used the free product of its device. Its tenant record is queued.
	ProductRequestGateFree ProductRequestGate = "free"
	// ProductRequestGatePaymentRequired means the request stays awaiting_gate until it is paid for (f.xiii). It
	// is the answer for a device that already claimed its free product, for a request that carried no usable
	// fingerprint, and for an instance without DEVICE_FINGERPRINT_SALT.
	ProductRequestGatePaymentRequired ProductRequestGate = "payment_required"
)
