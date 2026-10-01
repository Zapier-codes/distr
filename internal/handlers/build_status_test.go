package handlers

import (
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestBuildReportAlreadyApplied(t *testing.T) {
	asset := api.BuildReleaseAsset{Repository: "Zapier-codes/Storeapp", ReleaseID: 7, AssetID: 9}
	succeeded := api.BuildStatusRequest{
		TenantConfigID: uuid.New(), Status: types.TenantBuildStatusSucceeded, Asset: &asset,
	}
	failed := api.BuildStatusRequest{TenantConfigID: uuid.New(), Status: types.TenantBuildStatusFailed}
	current := func(status types.TenantBuildStatus, repo string, release, assetID int64) types.TenantConfig {
		config := types.TenantConfig{BuildStatus: status}
		if repo != "" {
			config.ReleaseRepository, config.ReleaseID, config.ReleaseAssetID = &repo, &release, &assetID
		}
		return config
	}

	g := NewWithT(t)
	g.Expect(buildReportAlreadyApplied(succeeded,
		current(types.TenantBuildStatusSucceeded, "Zapier-codes/Storeapp", 7, 9))).To(BeTrue())
	g.Expect(buildReportAlreadyApplied(succeeded,
		current(types.TenantBuildStatusSucceeded, "Zapier-codes/Storeapp", 7, 10))).To(BeFalse())
	g.Expect(buildReportAlreadyApplied(succeeded,
		current(types.TenantBuildStatusSucceeded, "Zapier-codes/Storeapp", 8, 9))).To(BeFalse())
	g.Expect(buildReportAlreadyApplied(succeeded, current(types.TenantBuildStatusSucceeded, "", 0, 0))).To(BeFalse())
	g.Expect(buildReportAlreadyApplied(succeeded, current(types.TenantBuildStatusFailed, "", 0, 0))).To(BeFalse())
	g.Expect(buildReportAlreadyApplied(succeeded, current(types.TenantBuildStatusBuilding, "", 0, 0))).To(BeFalse())
	g.Expect(buildReportAlreadyApplied(failed, current(types.TenantBuildStatusFailed, "", 0, 0))).To(BeTrue())
	g.Expect(buildReportAlreadyApplied(failed, current(types.TenantBuildStatusSucceeded, "a/b", 1, 2))).To(BeFalse())
}
