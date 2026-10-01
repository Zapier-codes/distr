package types

import (
	"time"

	"github.com/google/uuid"
)

type TenantBuildStatus string

const (
	// TenantBuildStatusAwaitingGate is the state of a new record. It stays there until the free/paid gate clears it.
	TenantBuildStatusAwaitingGate TenantBuildStatus = "awaiting_gate"
	TenantBuildStatusQueued       TenantBuildStatus = "queued"
	TenantBuildStatusBuilding     TenantBuildStatus = "building"
	TenantBuildStatusSucceeded    TenantBuildStatus = "succeeded"
	TenantBuildStatusFailed       TenantBuildStatus = "failed"
)

// TenantConfig is the canonical tenant record, in the shape of Storeapp's spec/tenant-config-schema.md (v1).
// It carries nothing about the requester.
type TenantConfig struct {
	ID                   uuid.UUID         `db:"id"`
	CreatedAt            time.Time         `db:"created_at"`
	TenantID             string            `db:"tenant_id"`
	ProductServiceID     uuid.UUID         `db:"product_service_id"`
	DisplayName          string            `db:"display_name"`
	PrimaryColorHex      string            `db:"primary_color_hex"`
	Logo                 []byte            `db:"logo"`
	LogoContentType      *string           `db:"logo_content_type"`
	LogoSHA256           *string           `db:"logo_sha256"`
	CDNBase              string            `db:"cdn_base"`
	CatalogIndexBaseURL  *string           `db:"catalog_index_base_url"`
	Domains              []string          `db:"domains"`
	Sequence             int               `db:"sequence"`
	BuildStatus          TenantBuildStatus `db:"build_status"`
	BuildStatusUpdatedAt time.Time         `db:"build_status_updated_at"`
}
