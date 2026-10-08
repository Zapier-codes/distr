package types

import (
	"testing"

	. "github.com/onsi/gomega"
)

// g.iii-a: the developer role is the lowest rank, so every vendor-portal role check (read_write/admin) refuses it
// without having to name it. This pins that ordering, which the whole authorization story rests on.
func TestDeveloperRoleRanksBelowEveryPortalRole(t *testing.T) {
	g := NewWithT(t)

	g.Expect(UserRoleDeveloper.GreaterThan(UserRoleReadOnly)).To(BeFalse())
	g.Expect(UserRoleDeveloper.GreaterThan(UserRoleReadWrite)).To(BeFalse())
	g.Expect(UserRoleDeveloper.GreaterThan(UserRoleAdmin)).To(BeFalse())
	g.Expect(UserRoleReadOnly.GreaterThan(UserRoleDeveloper)).To(BeTrue())
	g.Expect(UserRoleAdmin.GreaterThan(UserRoleDeveloper)).To(BeTrue())

	parsed, err := ParseUserRole("developer")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parsed).To(Equal(UserRoleDeveloper))
}
