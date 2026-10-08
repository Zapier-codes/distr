package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func ProductServiceToAPI(service types.ProductService) api.ProductService {
	var price *api.ProductPrice
	if service.PriceMinor != nil && service.PriceCurrency != nil {
		price = &api.ProductPrice{AmountMinor: *service.PriceMinor, Currency: *service.PriceCurrency}
	}
	return api.ProductService{
		ID:          service.ID,
		Slug:        service.Slug,
		Type:        service.Type,
		Name:        service.Name,
		Summary:     service.Summary,
		Description: service.Description,
		Price:       price,
		Media:       List(service.Media, ProductServiceMediaToAPI),
	}
}

func ProductServiceMediaToAPI(media types.ProductServiceMedia) api.ProductServiceMediaItem {
	return api.ProductServiceMediaItem{
		Kind:    media.Kind,
		URL:     media.URL,
		Caption: media.Caption,
	}
}

// DeveloperListingToAPI maps a listing for its owner (g.iii-b). It never leaves the handler for a storefront
// visitor: the public catalogue uses ProductServiceToAPI, which hides the owner and the draft state.
func DeveloperListingToAPI(listing types.ProductService) api.DeveloperListing {
	var price *api.ProductPrice
	if listing.PriceMinor != nil && listing.PriceCurrency != nil {
		price = &api.ProductPrice{AmountMinor: *listing.PriceMinor, Currency: *listing.PriceCurrency}
	}
	return api.DeveloperListing{
		ID:            listing.ID,
		Slug:          listing.Slug,
		Type:          listing.Type,
		Name:          listing.Name,
		Summary:       listing.Summary,
		Description:   listing.Description,
		Active:        listing.Active,
		ListingStatus: listing.ListingStatus,
		Price:         price,
	}
}
