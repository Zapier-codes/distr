package handlers

import (
	"testing"

	. "github.com/onsi/gomega"
)

// g.i-c: the free product must never be reachable without the human check (D4). The gate needs the fingerprint salt
// and a configured Turnstile; either one missing leaves every request awaiting_gate rather than giving it away free.
func TestFreeTierCheckConfigured(t *testing.T) {
	salt := "a-salt-of-at-least-32-characters-long"
	secret := "a-turnstile-secret"

	tests := []struct {
		name            string
		salt            *string
		turnstileSecret *string
		want            bool
	}{
		{"both set", &salt, &secret, true},
		{"no salt", nil, &secret, false},
		{"no turnstile", &salt, nil, false},
		{"neither set", nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(freeTierCheckConfigured(tt.salt, tt.turnstileSecret)).To(Equal(tt.want))
		})
	}
}
