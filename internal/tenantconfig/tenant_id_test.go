package tenantconfig_test

import (
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/tenantconfig"
)

func TestGenerateTenantID(t *testing.T) {
	tests := map[string]string{
		"Acme App Store":         "acme-app-store-",
		"  --Weird__Name!!  ":    "weird-name-",
		"Ünïcödé":                "n-c-d-",
		"!!!":                    "app-",
		strings.Repeat("a", 200): strings.Repeat("a", 40) + "-",
		"日本語":                    "app-",
	}
	for input, wantPrefix := range tests {
		got, err := tenantconfig.GenerateTenantID(input)
		if err != nil {
			t.Fatalf("GenerateTenantID(%q): %v", input, err)
		}
		if !strings.HasPrefix(got, wantPrefix) {
			t.Errorf("GenerateTenantID(%q) = %q, want prefix %q", input, got, wantPrefix)
		}
		if !tenantconfig.TenantIDPattern.MatchString(got) {
			t.Errorf("GenerateTenantID(%q) = %q is not a valid DNS label", input, got)
		}
	}
}

func TestGenerateTenantIDIsUnique(t *testing.T) {
	a, _ := tenantconfig.GenerateTenantID("same")
	b, _ := tenantconfig.GenerateTenantID("same")
	if a == b {
		t.Errorf("two ids for the same name are equal: %q", a)
	}
}
