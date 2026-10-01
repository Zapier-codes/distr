package api

import (
	"time"

	"github.com/google/uuid"
)

// BuildConfigSchemaVersion is the schema_version of Storeapp's spec/tenant-config-schema.md that BuildConfig follows.
const BuildConfigSchemaVersion = 1

// BuildConfig is what the tenant build workflow of Storeapp fetches. It is the tenant record in the shape of
// Storeapp's spec/tenant-config-schema.md (v1), so the field names are that document's, in snake_case, plus
// TenantConfigID, which that document does not have. Unrecognized fields are additive in that schema, so a reader of
// the schema alone ignores it.
//
// It carries nothing about the requester. The packaging identity of the APK (applicationId, launcher name and icon)
// is not in the schema; the workflow derives it from TenantID and Branding.
type BuildConfig struct {
	SchemaVersion int `json:"schema_version"`
	// TenantConfigID is the id the build was dispatched with. The build-status webhook (f.iv.zo) carries it back.
	TenantConfigID uuid.UUID `json:"tenant_config_id"`
	TenantID       string    `json:"tenant_id"`
	// GeneratedAt is when this answer was produced. The record is not a publish, so it has no publish time of its own.
	GeneratedAt time.Time `json:"generated_at"`
	// Sequence is 0 until the record has been published, see the schema.
	Sequence int `json:"sequence"`
	// ExpiresAt bounds how long a reader may rely on this answer. A build finishes long before it.
	ExpiresAt           time.Time           `json:"expires_at"`
	Branding            BuildConfigBranding `json:"branding"`
	CDNBase             string              `json:"cdn_base"`
	CatalogIndexBaseURL *string             `json:"catalog_index_base_url,omitempty"`
	Domains             []string            `json:"domains"`
	IsDefaultTenant     bool                `json:"is_default_tenant"`
}

type BuildConfigBranding struct {
	DisplayName     string `json:"display_name"`
	PrimaryColorHex string `json:"primary_color_hex"`
	// LogoURL is where the icon can be downloaded, without credentials. The workflow checks the bytes against
	// LogoSHA256. Both are absent when the requester gave no icon.
	LogoURL    *string `json:"logo_url,omitempty"`
	LogoSHA256 *string `json:"logo_sha256,omitempty"`
}
