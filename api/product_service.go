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
	Media       []ProductServiceMediaItem `json:"media"`
}

// ProductServiceMediaItem is a demo screenshot or video of a ProductService, given as a URL.
type ProductServiceMediaItem struct {
	Kind    types.ProductServiceMediaKind `json:"kind"`
	URL     string                        `json:"url"`
	Caption *string                       `json:"caption,omitempty"`
}
