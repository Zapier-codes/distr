package buildlink

import (
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

// Known answer, the same one Zealot's spec checks, so the two sides cannot drift apart:
//
//	printf 'tenant-build-download\nb123\n1900000000' | openssl dgst -sha256 -hmac 'test-link-secret'
func TestSignKnownAnswer(t *testing.T) {
	g := NewWithT(t)
	g.Expect(Sign("b123", 1900000000, "test-link-secret")).
		To(Equal("44cc1c406d7309d5ad224f7c840562a78fe56af535217bcb177a6ad00e03508f"))
}

func TestURL(t *testing.T) {
	g := NewWithT(t)
	id := uuid.MustParse("0b6a1a64-5f0d-4a52-9f0c-6a3f1f1e8a11")
	now := time.Unix(1900000000, 0)
	expires := int64(1900000600)

	got := URL("https://zealot.example.com", id, "test-link-secret", now, DefaultTTL)

	g.Expect(got).To(Equal(
		"https://zealot.example.com/api/tenant_builds/0b6a1a64-5f0d-4a52-9f0c-6a3f1f1e8a11/download" +
			"?expires=1900000600&signature=" + Sign(id.String(), expires, "test-link-secret")))
}

func TestSignDependsOnEveryInput(t *testing.T) {
	g := NewWithT(t)
	base := Sign("b123", 1900000000, "secret")
	g.Expect(Sign("b124", 1900000000, "secret")).NotTo(Equal(base))
	g.Expect(Sign("b123", 1900000001, "secret")).NotTo(Equal(base))
	g.Expect(Sign("b123", 1900000000, "other")).NotTo(Equal(base))
}
