package mapping_test

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestTenantConfigToBuildConfig(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	sha := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	base := types.TenantConfig{
		ID:              uuid.New(),
		TenantID:        "acme-store-a1b2c3",
		DisplayName:     "Acme Store",
		PrimaryColorHex: "#FF6600",
		CDNBase:         "https://cdn.example.com/metadata",
		Sequence:        0,
	}

	t.Run("with logo", func(t *testing.T) {
		g := NewWithT(t)
		tenant := base
		tenant.LogoSHA256 = &sha
		got := mapping.TenantConfigToBuildConfig(tenant, "https://distr.example.com/api/public/v1/tenant-logos/x", now)
		g.Expect(got.SchemaVersion).To(Equal(1))
		g.Expect(got.TenantConfigID).To(Equal(tenant.ID))
		g.Expect(got.Branding.LogoURL).To(HaveValue(Equal("https://distr.example.com/api/public/v1/tenant-logos/x")))
		g.Expect(got.Branding.LogoSHA256).To(HaveValue(Equal(sha)))
		g.Expect(got.GeneratedAt).To(Equal(now))
		g.Expect(got.ExpiresAt).To(Equal(now.Add(mapping.BuildConfigValidity)))
		g.Expect(got.IsDefaultTenant).To(BeFalse())
	})

	t.Run("without logo", func(t *testing.T) {
		g := NewWithT(t)
		got := mapping.TenantConfigToBuildConfig(base, "https://ignored.example.com/x", now)
		g.Expect(got.Branding.LogoURL).To(BeNil())
		g.Expect(got.Branding.LogoSHA256).To(BeNil())
	})

	t.Run("no domains is an empty list, not null", func(t *testing.T) {
		g := NewWithT(t)
		got := mapping.TenantConfigToBuildConfig(base, "", now)
		g.Expect(got.Domains).NotTo(BeNil())
		g.Expect(got.Domains).To(BeEmpty())
	})
}
