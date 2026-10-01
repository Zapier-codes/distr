package types

import (
	"time"

	"github.com/google/uuid"
)

type ProductServiceType string

const (
	ProductServiceTypeApp     ProductServiceType = "app"
	ProductServiceTypeWebsite ProductServiceType = "website"
)

type ProductServiceMediaKind string

const (
	ProductServiceMediaKindScreenshot ProductServiceMediaKind = "screenshot"
	ProductServiceMediaKindVideo      ProductServiceMediaKind = "video"
)

// ProductService is an entry of the public request storefront.
type ProductService struct {
	ID          uuid.UUID          `db:"id"`
	CreatedAt   time.Time          `db:"created_at"`
	Slug        string             `db:"slug"`
	Type        ProductServiceType `db:"type"`
	Name        string             `db:"name"`
	Summary     string             `db:"summary"`
	Description string             `db:"description"`
	Active      bool               `db:"active"`
	SortOrder   int                `db:"sort_order"`
	// Media is filled by a separate query, not by the row.
	Media []ProductServiceMedia `db:"-"`
}

type ProductServiceMedia struct {
	ID               uuid.UUID               `db:"id"`
	ProductServiceID uuid.UUID               `db:"product_service_id"`
	Kind             ProductServiceMediaKind `db:"kind"`
	URL              string                  `db:"url"`
	Caption          *string                 `db:"caption"`
	SortOrder        int                     `db:"sort_order"`
}
