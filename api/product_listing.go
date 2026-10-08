package api

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// DeveloperListing is a marketplace listing as its owner sees it (g.iii-b): the storefront fields plus the
// automatic-publish state and the USD price. It is never returned to a storefront visitor.
type DeveloperListing struct {
	ID            uuid.UUID                         `json:"id"`
	Slug          string                            `json:"slug"`
	Type          types.ProductServiceType          `json:"type"`
	Name          string                            `json:"name"`
	Summary       string                            `json:"summary"`
	Description   string                            `json:"description"`
	Active        bool                              `json:"active"`
	ListingStatus types.ProductServiceListingStatus `json:"listingStatus"`
	// Price is nil while the listing has none, and such a listing cannot be purchased.
	Price *ProductPrice `json:"price,omitempty"`
}

// CreateProductListingRequest is a developer's first listing (g.iii-b). The owner is taken from the session, never
// from the body, and the listing starts in the draft state.
type CreateProductListingRequest struct {
	Type        types.ProductServiceType `json:"type"`
	Name        string                   `json:"name"`
	Summary     string                   `json:"summary"`
	Description string                   `json:"description"`
	// PriceAmountMinor and PriceCurrency are optional together; currency must be USD (g.ii, D5).
	PriceAmountMinor *int    `json:"priceAmountMinor,omitempty"`
	PriceCurrency    *string `json:"priceCurrency,omitempty"`
}

// UpdateProductListingRequest edits a listing the caller owns. A nil field is left unchanged; a non-nil field is
// written as given. The owner and the slug cannot be changed here.
type UpdateProductListingRequest struct {
	Name             *string                            `json:"name,omitempty"`
	Summary          *string                            `json:"summary,omitempty"`
	Description      *string                            `json:"description,omitempty"`
	ListingStatus    *types.ProductServiceListingStatus `json:"listingStatus,omitempty"`
	PriceAmountMinor *int                               `json:"priceAmountMinor,omitempty"`
	PriceCurrency    *string                            `json:"priceCurrency,omitempty"`
	// ClearPrice removes the price when true, so a listing can go back to being free.
	ClearPrice bool `json:"clearPrice,omitempty"`
}
