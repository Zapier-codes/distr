// Package buildlink signs the short-lived link to Zealot's download door (task 40n-g).
//
// The mail a requester gets carries a random token that opens distr's own route. When Zealot serves the binary
// (ZEALOT_URL and DISTR_LINK_SECRET are set), that route does not hand out anything itself: at click time it mints a
// fresh link to Zealot's door and redirects to it. Nothing long-lived is ever signed, so a mail that is forwarded or
// scanned a month later holds no usable download, and the shared secret never leaves the two servers.
//
// A link is Zealot's door plus two query parameters:
//
//	<zealot>/api/tenant_builds/<build_id>/download?expires=<unix seconds>&signature=<64 hex>
//
// signature = HMAC-SHA256(secret, "tenant-build-download\n<build_id>\n<expires>"), lowercase hex. Zealot's door
// (app/services/tenant_build_link.rb) recomputes it, compares in constant time, and refuses an expired link or one
// that is further out than seven days. The build id is the id of the TenantConfig record, the id the build was
// dispatched with.
package buildlink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// Context separates this signature from any other use of the same secret. Zealot uses the same value.
	Context = "tenant-build-download"
	// DefaultTTL is how long a minted link works. It only has to outlive one redirect, so it is short.
	DefaultTTL = 10 * time.Minute
)

// Sign returns the lowercase hex signature of a build id and an expiry in unix seconds.
func Sign(buildID string, expires int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s\n%s\n%d", Context, buildID, expires)
	return hex.EncodeToString(mac.Sum(nil))
}

// URL returns the signed link to Zealot's door for a build, valid for ttl from now. baseURL is Zealot's origin
// without a trailing slash.
func URL(baseURL string, buildID uuid.UUID, secret string, now time.Time, ttl time.Duration) string {
	expires := now.Add(ttl).Unix()
	return fmt.Sprintf("%s/api/tenant_builds/%s/download?expires=%d&signature=%s",
		baseURL, buildID, expires, Sign(buildID.String(), expires, secret))
}
