package buildnotify

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewToken(t *testing.T) {
	g := NewWithT(t)
	token, hash, err := NewToken()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(token).To(HaveLen(TokenLength))
	g.Expect(token).To(MatchRegexp(`^[A-Za-z0-9_-]+$`))
	g.Expect(hash).To(HaveLen(32))
	g.Expect(HashToken(token)).To(Equal(hash))

	other, otherHash, err := NewToken()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(other).NotTo(Equal(token))
	g.Expect(otherHash).NotTo(Equal(hash))
}

func TestDownloadURL(t *testing.T) {
	g := NewWithT(t)
	g.Expect(DownloadURL("https://distr.example.com", "abc")).
		To(Equal("https://distr.example.com/api/public/v1/build-downloads/abc"))
}

func TestTestSubscriberID(t *testing.T) {
	g := NewWithT(t)
	id := testSubscriberID("Someone@Example.com ")
	g.Expect(id).To(MatchRegexp(`^distr-test-[0-9a-f]{16}$`))
	g.Expect(testSubscriberID("someone@example.com")).To(Equal(id))
	g.Expect(testSubscriberID("other@example.com")).NotTo(Equal(id))
}

func TestTestPayload(t *testing.T) {
	g := NewWithT(t)
	g.Expect(testPayload("https://distr.example.com")).To(Equal(map[string]string{
		"appName":     "Sample app",
		"downloadUrl": "https://distr.example.com/store",
	}))
}
