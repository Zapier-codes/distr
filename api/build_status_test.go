package api

import (
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestBuildStatusRequestValidate(t *testing.T) {
	asset := &BuildReleaseAsset{Repository: "Zapier-codes/Storeapp", ReleaseID: 1, AssetID: 2}
	long := strings.Repeat("x", MaxBuildStatusMessageLength+1)
	id := uuid.New()
	succeeded := types.TenantBuildStatusSucceeded
	failed := types.TenantBuildStatusFailed
	building := types.TenantBuildStatusBuilding
	tests := []struct {
		name    string
		request BuildStatusRequest
		valid   bool
	}{
		{"succeeded", BuildStatusRequest{TenantConfigID: id, Status: succeeded, Asset: asset}, true},
		{"succeeded without asset", BuildStatusRequest{TenantConfigID: id, Status: succeeded}, false},
		{"succeeded served by Zealot", BuildStatusRequest{TenantConfigID: id, Status: succeeded, ExternalAsset: true}, true},
		{"failed", BuildStatusRequest{TenantConfigID: id, Status: failed, Message: new("boom")}, true},
		{"failed without message", BuildStatusRequest{TenantConfigID: id, Status: failed}, true},
		{"failed message too long", BuildStatusRequest{TenantConfigID: id, Status: failed, Message: &long}, false},
		{"building is not final", BuildStatusRequest{TenantConfigID: id, Status: building, Asset: asset}, false},
		{"empty status", BuildStatusRequest{TenantConfigID: id, Asset: asset}, false},
		{"nil id", BuildStatusRequest{Status: failed}, false},
		{
			"repository is a URL",
			BuildStatusRequest{
				TenantConfigID: id, Status: succeeded,
				Asset: &BuildReleaseAsset{Repository: "https://github.com/a/b", ReleaseID: 1, AssetID: 2},
			},
			false,
		},
		{
			"zero release id",
			BuildStatusRequest{
				TenantConfigID: id, Status: succeeded,
				Asset: &BuildReleaseAsset{Repository: "a/b", ReleaseID: 0, AssetID: 2},
			},
			false,
		},
		{
			"zero asset id",
			BuildStatusRequest{
				TenantConfigID: id, Status: succeeded,
				Asset: &BuildReleaseAsset{Repository: "a/b", ReleaseID: 1, AssetID: 0},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			err := tt.request.Validate()
			if tt.valid {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}
