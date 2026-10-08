package env

import (
	"testing"

	. "github.com/onsi/gomega"
)

// g.i-c: Turnstile is all-or-nothing. A half-configured pair is a startup error, never a silent bypass, because the
// free product must not be reachable unverified (D4).
func TestParseTurnstile(t *testing.T) {
	t.Run("both unset means not configured, not an error", func(t *testing.T) {
		g := NewWithT(t)
		key, secret, err := ParseTurnstile("", "")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(key).To(BeNil())
		g.Expect(secret).To(BeNil())
	})

	t.Run("both set are returned", func(t *testing.T) {
		g := NewWithT(t)
		key, secret, err := ParseTurnstile("site-key", "secret-key")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(key).NotTo(BeNil())
		g.Expect(*key).To(Equal("site-key"))
		g.Expect(secret).NotTo(BeNil())
		g.Expect(*secret).To(Equal("secret-key"))
	})

	t.Run("a half-configured pair is refused, whichever half is missing", func(t *testing.T) {
		g := NewWithT(t)
		key, secret, err := ParseTurnstile("site-key", "")
		g.Expect(err).To(MatchError(ContainSubstring("must both be set, or neither")))
		g.Expect(key).To(BeNil())
		g.Expect(secret).To(BeNil())

		key, secret, err = ParseTurnstile("", "secret-key")
		g.Expect(err).To(MatchError(ContainSubstring("must both be set, or neither")))
		g.Expect(key).To(BeNil())
		g.Expect(secret).To(BeNil())
	})
}
