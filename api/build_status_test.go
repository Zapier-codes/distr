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
	tests := []struct {
		name    string
		request BuildStatusRequest
		valid   bool
	}{
		{"succeeded", BuildStatusRequest{uuid.New(), types.TenantBuildStatusSucceeded, nil, asset}, true},
		{"succeeded without asset", BuildStatusRequest{uuid.New(), types.TenantBuildStatusSucceeded, nil, nil}, false},
		{"failed", BuildStatusRequest{uuid.New(), types.TenantBuildStatusFailed, new("boom"), nil}, true},
		{"failed without message", BuildStatusRequest{uuid.New(), types.TenantBuildStatusFailed, nil, nil}, true},
		{"failed message too long", BuildStatusRequest{uuid.New(), types.TenantBuildStatusFailed, &long, nil}, false},
		{"building is not a final status", BuildStatusRequest{uuid.New(), types.TenantBuildStatusBuilding, nil, asset}, false},
		{"empty status", BuildStatusRequest{uuid.New(), "", nil, asset}, false},
		{"nil id", BuildStatusRequest{uuid.Nil, types.TenantBuildStatusFailed, nil, nil}, false},
		{"repository is a URL", BuildStatusRequest{uuid.New(), types.TenantBuildStatusSucceeded, nil,
			&BuildReleaseAsset{Repository: "https://github.com/a/b", ReleaseID: 1, AssetID: 2}}, false},
		{"zero release id", BuildStatusRequest{uuid.New(), types.TenantBuildStatusSucceeded, nil,
			&BuildReleaseAsset{Repository: "a/b", ReleaseID: 0, AssetID: 2}}, false},
		{"zero asset id", BuildStatusRequest{uuid.New(), types.TenantBuildStatusSucceeded, nil,
			&BuildReleaseAsset{Repository: "a/b", ReleaseID: 1, AssetID: 0}}, false},
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
