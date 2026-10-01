// Package tenantconfig holds the rules for the canonical tenant record that do not belong to a single layer.
package tenantconfig

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

const (
	maxSlugLength     = 40
	randomSuffixBytes = 3
	fallbackSlug      = "app"
)

var (
	nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)
	// TenantIDPattern is the DNS-label rule of the tenant_id field.
	TenantIDPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// GenerateTenantID derives a tenant_id from the display name and appends a random suffix. The value is permanent
// once a record is created, so the suffix, not the name, is what keeps it unique across renames and duplicates.
// The result is always a valid DNS label.
func GenerateTenantID(displayName string) (string, error) {
	slug := strings.Trim(nonSlugCharacters.ReplaceAllString(strings.ToLower(displayName), "-"), "-")
	if len(slug) > maxSlugLength {
		slug = strings.Trim(slug[:maxSlugLength], "-")
	}
	if slug == "" {
		slug = fallbackSlug
	}
	suffix := make([]byte, randomSuffixBytes)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	return slug + "-" + hex.EncodeToString(suffix), nil
}
