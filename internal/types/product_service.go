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

// ProductServiceListingStatus is the automatic-publish state of a listing (g.iii-b, D23). A listing is created
// `draft` and becomes purchasable only once it is `live`; `suspended` is the operator taking it down. The
// storefront's own entries (no owner) are `live` from the start.
type ProductServiceListingStatus string

const (
	ProductServiceListingStatusDraft     ProductServiceListingStatus = "draft"
	ProductServiceListingStatusLive      ProductServiceListingStatus = "live"
	ProductServiceListingStatusSuspended ProductServiceListingStatus = "suspended"
)

// PriceCurrencyUSD is the only currency a developer listing may be priced in (g.ii, D5): prices are USD cents.
const PriceCurrencyUSD = "USD"

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
	// OwnerUserAccountID is the developer who owns the listing (g.iii-b), or nil for the storefront's own entries.
	OwnerUserAccountID *uuid.UUID                  `db:"owner_user_account_id"`
	ListingStatus      ProductServiceListingStatus `db:"listing_status"`
	// PriceMinor and PriceCurrency are the one-time price in the minor unit of the currency, both nil while the
	// product has no price. A product without a price cannot be paid for.
	PriceMinor    *int    `db:"price_minor"`
	PriceCurrency *string `db:"price_currency"`
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
