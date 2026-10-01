package buildtrigger

import (
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
)

func TestResolveAssetDownloadReturnsLocationWithoutFollowing(t *testing.T) {
	g := NewWithT(t)
	signed := "https://release-assets.githubusercontent.com/x?sig=abc"
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodGet))
		g.Expect(r.URL.Path).To(Equal("/repos/Zapier-codes/Storeapp/releases/assets/42"))
		g.Expect(r.Header.Get("Authorization")).To(Equal("Bearer test-token"))
		g.Expect(r.Header.Get("Accept")).To(Equal("application/octet-stream"))
		w.Header().Set("Location", signed)
		w.WriteHeader(http.StatusFound)
	}))
	got, err := client.ResolveAssetDownload(t.Context(), "Zapier-codes/Storeapp", 42)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(signed))
}

func TestResolveAssetDownloadNotFound(t *testing.T) {
	g := NewWithT(t)
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	_, err := client.ResolveAssetDownload(t.Context(), "Zapier-codes/Storeapp", 1)
	g.Expect(err).To(MatchError(ErrAssetNotFound))
}

func TestResolveAssetDownloadRefusesForeignLocation(t *testing.T) {
	for _, location := range []string{
		"http://objects.githubusercontent.com/x",
		"https://evil.example.com/x",
		"https://githubusercontent.com.evil.example.com/x",
		"https://user@objects.githubusercontent.com/x",
		"",
	} {
		t.Run(location, func(t *testing.T) {
			g := NewWithT(t)
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if location != "" {
					w.Header().Set("Location", location)
				}
				w.WriteHeader(http.StatusFound)
			}))
			_, err := client.ResolveAssetDownload(t.Context(), "Zapier-codes/Storeapp", 1)
			g.Expect(err).To(HaveOccurred())
		})
	}
}
