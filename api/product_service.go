package api

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// ProductService is an entry of the public request storefront.
type ProductService struct {
	ID          uuid.UUID                 `json:"id"`
	Slug        string                    `json:"slug"`
	Type        types.ProductServiceType  `json:"type"`
	Name        string                    `json:"name"`
	Summary     string                    `json:"summary"`
	Description string                    `json:"description"`
	// Price is nil while the product has none, and such a product cannot be paid for.
	Price *ProductPrice             `json:"price,omitempty"`
	Media []ProductServiceMediaItem `json:"media"`
}

// ProductPrice is a one-time price. AmountMinor is in the minor unit of the currency (kobo, cents), Currency is an
// ISO 4217 code.
type ProductPrice struct {
	AmountMinor int    `json:"amountMinor"`
	Currency    string `json:"currency"`
}

// ProductServiceMediaItem is a demo screenshot or video of a ProductService, given as a URL.
type ProductServiceMediaItem struct {
	Kind    types.ProductServiceMediaKind `json:"kind"`
	URL     string                        `json:"url"`
	Caption *string                       `json:"caption,omitempty"`
}
