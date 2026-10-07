package buildtrigger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/distr-sh/distr/internal/env"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

var testConfig = env.StoreappBuildConfig{
	Token:      "test-token",
	Repository: "Zapier-codes/Storeapp",
	Workflow:   "build-tenant-apk.yml",
	Ref:        "main",
}

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(testConfig, server.URL)
}

func TestDispatchSendsCanonicalRequest(t *testing.T) {
	g := NewWithT(t)
	id := uuid.New()

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodPost))
		g.Expect(r.URL.Path).To(Equal("/repos/Zapier-codes/Storeapp/actions/workflows/build-tenant-apk.yml/dispatches"))
		g.Expect(r.Header.Get("Authorization")).To(Equal("Bearer test-token"))
		g.Expect(r.Header.Get("Accept")).To(Equal("application/vnd.github+json"))
		g.Expect(r.Header.Get("X-GitHub-Api-Version")).To(Equal("2022-11-28"))

		var body dispatchRequest
		g.Expect(json.NewDecoder(r.Body).Decode(&body)).To(Succeed())
		g.Expect(body.Ref).To(Equal("main"))
		g.Expect(body.Inputs).To(Equal(map[string]string{"build_id": id.String()}))
		w.WriteHeader(http.StatusNoContent)
	}))

	g.Expect(client.Dispatch(t.Context(), id)).To(Succeed())
}

func TestDispatchReportsGitHubsReason(t *testing.T) {
	g := NewWithT(t)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Workflow does not have 'workflow_dispatch' trigger"}`))
	}))

	err := client.Dispatch(t.Context(), uuid.New())
	g.Expect(err).To(MatchError(ContainSubstring("status 422")))
	g.Expect(err).To(MatchError(ContainSubstring("workflow_dispatch")))
}

func TestDispatchBoundsGitHubsMessageAndKeepsTheTokenOut(t *testing.T) {
	g := NewWithT(t)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"` + strings.Repeat("x", 5000) + `"}`))
	}))

	err := client.Dispatch(t.Context(), uuid.New())
	g.Expect(err).To(HaveOccurred())
	g.Expect(len(err.Error())).To(BeNumerically("<", 300))
	g.Expect(err.Error()).NotTo(ContainSubstring("test-token"))
}

func TestDispatchToleratesAnUnreadableErrorBody(t *testing.T) {
	g := NewWithT(t)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`<html>bad gateway</html>`))
	}))

	g.Expect(client.Dispatch(t.Context(), uuid.New())).To(MatchError("GitHub refused the dispatch with status 502"))
}

func TestDispatchEscapesTheWorkflowName(t *testing.T) {
	g := NewWithT(t)

	config := testConfig
	config.Workflow = "a b/../c.yml"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.EscapedPath()).To(
			Equal("/repos/Zapier-codes/Storeapp/actions/workflows/a%20b%2F..%2Fc.yml/dispatches"))
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	g.Expect(NewClient(config, server.URL).Dispatch(t.Context(), uuid.New())).To(Succeed())
}

func TestDispatchCutsTheMessageOnARuneBoundary(t *testing.T) {
	g := NewWithT(t)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"` + strings.Repeat("é", 500) + `"}`))
	}))

	err := client.Dispatch(t.Context(), uuid.New())
	g.Expect(err).To(HaveOccurred())
	g.Expect(utf8.ValidString(err.Error())).To(BeTrue())
}
