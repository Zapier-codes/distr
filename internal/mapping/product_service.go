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
