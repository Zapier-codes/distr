package mapping

import (
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

// BuildConfigValidity is how long ExpiresAt of a BuildConfig is after GeneratedAt.
const BuildConfigValidity = 24 * time.Hour

// TenantConfigToBuildConfig converts the record to the answer of the build-config endpoint. logoURL is only used
// when the record has a logo, and is the absolute URL of its public download.
func TenantConfigToBuildConfig(tenant types.TenantConfig, logoURL string, now time.Time) api.BuildConfig {
	branding := api.BuildConfigBranding{
		DisplayName:     tenant.DisplayName,
		PrimaryColorHex: tenant.PrimaryColorHex,
	}
	if tenant.LogoSHA256 != nil {
		branding.LogoURL = &logoURL
		branding.LogoSHA256 = tenant.LogoSHA256
	}

	domains := tenant.Domains
	if domains == nil {
		domains = []string{}
	}

	now = now.UTC()
	return api.BuildConfig{
		SchemaVersion:       api.BuildConfigSchemaVersion,
		TenantConfigID:      tenant.ID,
		TenantID:            tenant.TenantID,
		GeneratedAt:         now,
		Sequence:            tenant.Sequence,
		ExpiresAt:           now.Add(BuildConfigValidity),
		Branding:            branding,
		CDNBase:             tenant.CDNBase,
		CatalogIndexBaseURL: tenant.CatalogIndexBaseURL,
		Domains:             domains,
		IsDefaultTenant:     false,
	}
}
