package api

import (
	"regexp"
	"unicode/utf8"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

// MaxBuildStatusMessageLength bounds the reason of a failed build that is kept, in characters.
const MaxBuildStatusMessageLength = 500

var buildReleaseRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// BuildStatusRequest is what the tenant build workflow of Storeapp posts when its run ends.
type BuildStatusRequest struct {
	// TenantConfigID is the id the build was dispatched with, which the build-config answer carried back.
	TenantConfigID uuid.UUID `json:"tenant_config_id"`
	// BuildID is the same id under the name Zealot's harvest workflow reports it by. Either field may carry it; the
	// handler copies it into TenantConfigID when that one is empty.
	BuildID uuid.UUID `json:"build_id"`
	// Status is succeeded or failed.
	Status types.TenantBuildStatus `json:"status"`
	// Message says why a build failed. It is ignored for a succeeded build.
	Message *string `json:"message,omitempty"`
	// Asset identifies the GitHub Release asset of a succeeded build. It is required then, and ignored otherwise.
	Asset *BuildReleaseAsset `json:"asset,omitempty"`
	// ExternalAsset is set by the handler, never read from the request: it says the binary of a succeeded build is
	// served by Zealot, so no release asset is expected.
	ExternalAsset bool `json:"-"`
}

// BuildReleaseAsset is a GitHub Release asset by identifiers. It is never a URL: a private repository's asset cannot
// be fetched from one, and the identifiers are what the download redirect resolves at click time.
type BuildReleaseAsset struct {
	// Repository is "owner/name".
	Repository string `json:"repository"`
	ReleaseID  int64  `json:"release_id"`
	AssetID    int64  `json:"asset_id"`
}

func (r *BuildStatusRequest) Validate() error {
	if r.TenantConfigID == uuid.Nil {
		return validation.NewValidationFailedError("tenant_config_id is empty")
	}
	switch r.Status {
	case types.TenantBuildStatusFailed:
		if r.Message != nil && utf8.RuneCountInString(*r.Message) > MaxBuildStatusMessageLength {
			return validation.NewValidationFailedError("message is too long")
		}
		return nil
	case types.TenantBuildStatusSucceeded:
		if r.Asset == nil {
			if r.ExternalAsset {
				return nil
			}
			return validation.NewValidationFailedError("a succeeded build needs the release asset")
		}
		return r.Asset.Validate()
	default:
		return validation.NewValidationFailedError("status must be succeeded or failed")
	}
}

func (a BuildReleaseAsset) Validate() error {
	if !buildReleaseRepositoryPattern.MatchString(a.Repository) {
		return validation.NewValidationFailedError("asset.repository must be owner/name")
	}
	if a.ReleaseID <= 0 {
		return validation.NewValidationFailedError("asset.release_id must be positive")
	}
	if a.AssetID <= 0 {
		return validation.NewValidationFailedError("asset.asset_id must be positive")
	}
	return nil
}
