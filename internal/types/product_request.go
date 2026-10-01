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
