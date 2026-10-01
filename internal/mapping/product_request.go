package mapping

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func ProductRequestToAPI(request types.ProductRequest, tenant types.TenantConfig) api.ProductRequest {
	return api.ProductRequest{
		ID:          request.ID,
		CreatedAt:   request.CreatedAt,
		TenantID:    tenant.TenantID,
		BuildStatus: tenant.BuildStatus,
	}
}

func ProductRequestToInternal(request api.CreateProductRequestRequest) types.ProductRequest {
	return types.ProductRequest{
		ContactEmail: request.ContactEmail,
		AppName:      request.AppName,
		ThemeColor:   request.ThemeColor,
	}
}

// ProductRequestToTenantConfig is the branding half of a request as a tenant record. The contact email is
// deliberately not carried over. TenantID is left for the caller, which has to make it unique.
func ProductRequestToTenantConfig(request api.CreateProductRequestRequest) types.TenantConfig {
	tenant := types.TenantConfig{
		ProductServiceID: request.ProductServiceID,
		DisplayName:      request.AppName,
		PrimaryColorHex:  request.ThemeColor,
	}
	if request.Icon != nil {
		sum := sha256.Sum256(request.Icon.Data)
		tenant.Logo = request.Icon.Data
		tenant.LogoContentType = &request.Icon.ContentType
		tenant.LogoSHA256 = new(hex.EncodeToString(sum[:]))
	}
	return tenant
}
