package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/distr-sh/distr/internal/env"
	. "github.com/onsi/gomega"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		token  string
		ok     bool
	}{
		{"valid", "Bearer abc123", "abc123", true},
		{"scheme is case-insensitive", "bEaReR abc123", "abc123", true},
		{"surrounding whitespace is ignored", "Bearer   abc123  ", "abc123", true},
		{"missing header", "", "", false},
		{"other scheme", "Basic abc123", "", false},
		{"scheme without token", "Bearer ", "", false},
		{"token without scheme", "abc123", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			token, ok := bearerToken(req)
			g.Expect(ok).To(Equal(tt.ok))
			g.Expect(token).To(Equal(tt.token))
		})
	}
}

func TestBearerTokenMatches(t *testing.T) {
	g := NewWithT(t)
	expected := "0123456789abcdef0123456789abcdef"
	g.Expect(bearerTokenMatches(expected, expected)).To(BeTrue())
	g.Expect(bearerTokenMatches("", expected)).To(BeFalse())
	g.Expect(bearerTokenMatches(expected+"x", expected)).To(BeFalse())
	g.Expect(bearerTokenMatches(expected[:len(expected)-1], expected)).To(BeFalse())
	g.Expect(bearerTokenMatches("0123456789abcdef0123456789abcdeF", expected)).To(BeFalse())
}

func TestPublicBaseURL(t *testing.T) {
	g := NewWithT(t)
	g.Expect(publicBaseURL("distr.example.com", env.SchemeHTTPS)).To(Equal("https://distr.example.com"))
	g.Expect(publicBaseURL("localhost:8080", env.SchemeHTTP)).To(Equal("http://localhost:8080"))
	g.Expect(publicBaseURL(" distr.example.com/ ", env.SchemeHTTPS)).To(Equal("https://distr.example.com"))
	g.Expect(publicBaseURL("http://localhost:8080/", env.SchemeHTTP)).To(Equal("http://localhost:8080"))
	g.Expect(publicBaseURL("https://distr.example.com", env.SchemeHTTPS)).To(Equal("https://distr.example.com"))
}
