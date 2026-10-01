package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func ProductRequestToAPI(request types.ProductRequest) api.ProductRequest {
	return api.ProductRequest{
		ID:        request.ID,
		CreatedAt: request.CreatedAt,
	}
}

func ProductRequestToInternal(request api.CreateProductRequestRequest) types.ProductRequest {
	return types.ProductRequest{
		ContactEmail: request.ContactEmail,
		AppName:      request.AppName,
		ThemeColor:   request.ThemeColor,
	}
}
