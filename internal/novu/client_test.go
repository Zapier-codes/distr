package novu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/distr-sh/distr/internal/env"
	. "github.com/onsi/gomega"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(env.NovuConfig{APIKey: "secret-key", APIURL: server.URL, BuildReadyWorkflowID: "wf"})
}

func TestTriggerSendsCanonicalRequest(t *testing.T) {
	g := NewWithT(t)
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodPost))
		g.Expect(r.URL.Path).To(Equal("/v1/events/trigger"))
		g.Expect(r.Header.Get("Authorization")).To(Equal("ApiKey secret-key"))
		var body map[string]any
		g.Expect(json.NewDecoder(r.Body).Decode(&body)).To(Succeed())
		g.Expect(body["name"]).To(Equal("tenant-build-ready"))
		g.Expect(body["to"]).To(Equal(map[string]any{"subscriberId": "sub-1", "email": "a@example.com"}))
		g.Expect(body["payload"]).To(Equal(map[string]any{"downloadUrl": "https://x/y"}))
		w.WriteHeader(http.StatusCreated)
	})
	g.Expect(client.Trigger(t.Context(), "tenant-build-ready", "sub-1", "a@example.com",
		map[string]string{"downloadUrl": "https://x/y"})).To(Succeed())
}

func TestTriggerReportsRefusalWithoutKey(t *testing.T) {
	g := NewWithT(t)
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"workflow not found"}`))
	})
	err := client.Trigger(t.Context(), "missing", "sub", "a@example.com", nil)
	g.Expect(err).To(MatchError(ContainSubstring("status 400: workflow not found")))
	g.Expect(err.Error()).NotTo(ContainSubstring("secret-key"))
}

func TestTriggerRefusalWithListMessage(t *testing.T) {
	g := NewWithT(t)
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":["to.email must be an email"]}`))
	})
	err := client.Trigger(t.Context(), "wf", "sub", "bad", nil)
	g.Expect(err).To(MatchError(ContainSubstring("to.email must be an email")))
}
