package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

// testAuthInfo is the smallest identity that satisfies the authenticated-user interface. It embeds the interface
// so only the methods RequireNonDeveloper actually reads have to be implemented; any other call panics, which is
// what we want from a test stub.
type testAuthInfo struct {
	authinfo.AuthInfo
	role       *types.UserRole
	superAdmin bool
}

func (i testAuthInfo) CurrentUserRole() *types.UserRole                        { return i.role }
func (i testAuthInfo) IsSuperAdmin() bool                                      { return i.superAdmin }
func (i testAuthInfo) CurrentOrg() *types.Organization                         { return nil }
func (i testAuthInfo) CurrentOrgWithBranding() *types.OrganizationWithBranding { return nil }
func (i testAuthInfo) CurrentUser() *types.UserAccount                         { return nil }

func requestWithAuth(role *types.UserRole, isSuperAdmin bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil)
	ctx := auth.Authentication.NewContext(req.Context(), testAuthInfo{role: role, superAdmin: isSuperAdmin})
	return req.WithContext(ctx)
}

// g.iii-a: the backstop refuses the marketplace developer role on every route it wraps, passes the portal roles,
// and lets a super admin through (a super admin has no role, and is not a developer).
func TestRequireNonDeveloper(t *testing.T) {
	g := NewWithT(t)

	reached := false
	handler := RequireNonDeveloper(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	developer := types.UserRoleDeveloper
	readOnly := types.UserRoleReadOnly
	admin := types.UserRoleAdmin

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithAuth(&developer, false))
	g.Expect(rec.Code).To(Equal(http.StatusForbidden))
	g.Expect(reached).To(BeFalse())

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithAuth(&readOnly, false))
	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(reached).To(BeTrue())

	reached = false
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithAuth(&admin, false))
	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(reached).To(BeTrue())

	reached = false
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithAuth(nil, true))
	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(reached).To(BeTrue())
}

// Without an authenticated identity the middleware refuses, matching the other role middlewares.
func TestRequireNonDeveloperWithoutAuth(t *testing.T) {
	g := NewWithT(t)

	handler := RequireNonDeveloper(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil))
	g.Expect(rec.Code).To(Equal(http.StatusForbidden))
}
