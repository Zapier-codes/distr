package bpay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/distr-sh/distr/internal/env"
	. "github.com/onsi/gomega"
)

func signed(secret string, body []byte) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	g := NewWithT(t)
	body := []byte(`{"event_type":"payment_succeeded"}`)
	secret := "0123456789abcdef-secret"
	good := signed(secret, body)

	g.Expect(VerifySignature(secret, body, good)).To(BeTrue())
	g.Expect(VerifySignature(secret, body, " "+good+"\n")).To(BeTrue())
	g.Expect(VerifySignature("other-secret-0123456789", body, good)).To(BeFalse())
	g.Expect(VerifySignature(secret, append(body, ' '), good)).To(BeFalse())
	g.Expect(VerifySignature(secret, body, "")).To(BeFalse())
	g.Expect(VerifySignature(secret, body, "zz")).To(BeFalse())
	g.Expect(VerifySignature(secret, body, good[:len(good)-2])).To(BeFalse())
}

func TestParseWebhookEvent(t *testing.T) {
	g := NewWithT(t)
	event, err := ParseWebhookEvent([]byte(
		`{"merchant_id":"m","event_id":"e","event_type":"payment_succeeded",` +
			`"content":{"type":"payment_details","object":{"payment_id":"pay_123","status":"succeeded"}}}`))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(event.EventType).To(Equal(EventPaymentSucceeded))
	g.Expect(event.Content.Object.PaymentID).To(Equal("pay_123"))

	_, err = ParseWebhookEvent([]byte(`not json`))
	g.Expect(err).To(HaveOccurred())
}

func TestCreatePayment(t *testing.T) {
	g := NewWithT(t)
	var gotBody map[string]any
	var gotKey, gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotPath, gotMethod = r.Header.Get("api-key"), r.URL.Path, r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"payment_id":"pay_1","status":"requires_payment_method","amount":500000,` +
			`"currency":"NGN","payment_link":{"link":"https://pay.example.com/l/1"}}`))
	}))
	defer server.Close()

	profile := "pro_1"
	client := NewClient(env.BPayConfig{APIURL: server.URL, APIKey: "key", WebhookSecret: "x", ProfileID: &profile})
	payment, err := client.CreatePayment(t.Context(), 500000, "NGN", "https://distr.example.com/store", "Branded app",
		map[string]string{"distr_tenant_config_id": "abc"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(payment.ID).To(Equal("pay_1"))
	g.Expect(payment.Link).To(Equal("https://pay.example.com/l/1"))
	g.Expect(gotKey).To(Equal("key"))
	g.Expect(gotMethod).To(Equal(http.MethodPost))
	g.Expect(gotPath).To(Equal("/payments"))
	g.Expect(gotBody["amount"]).To(BeEquivalentTo(500000))
	g.Expect(gotBody["currency"]).To(Equal("NGN"))
	g.Expect(gotBody["payment_link"]).To(BeTrue())
	g.Expect(gotBody["profile_id"]).To(Equal("pro_1"))
}

func TestCreatePaymentRejectsAnswerWithoutLink(t *testing.T) {
	g := NewWithT(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"payment_id":"pay_1","status":"requires_payment_method","amount":1,"currency":"NGN"}`))
	}))
	defer server.Close()
	client := NewClient(env.BPayConfig{APIURL: server.URL, APIKey: "key", WebhookSecret: "x"})
	_, err := client.CreatePayment(t.Context(), 1, "NGN", "", "", nil)
	g.Expect(err).To(HaveOccurred())
}

func TestGetPayment(t *testing.T) {
	g := NewWithT(t)
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"payment_id":"pay_1","status":"succeeded","amount":500000,"currency":"ngn"}`))
	}))
	defer server.Close()
	client := NewClient(env.BPayConfig{APIURL: server.URL, APIKey: "key", WebhookSecret: "x"})

	payment, err := client.GetPayment(t.Context(), "pay_1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gotPath).To(Equal("/payments/pay_1"))
	g.Expect(payment.Status).To(Equal(StatusSucceeded))
	g.Expect(payment.AmountMinor).To(BeEquivalentTo(500000))
	g.Expect(payment.Currency).To(Equal("NGN"))

	_, err = client.GetPayment(t.Context(), "../x")
	g.Expect(err).To(HaveOccurred())
}

func TestRefusalMessageIsBounded(t *testing.T) {
	g := NewWithT(t)
	long := make([]byte, 1000)
	for i := range long {
		long[i] = 'a'
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"` + string(long) + `"}}`))
	}))
	defer server.Close()
	client := NewClient(env.BPayConfig{APIURL: server.URL, APIKey: "secret-key", WebhookSecret: "x"})
	_, err := client.GetPayment(t.Context(), "pay_1")
	g.Expect(err).To(HaveOccurred())
	g.Expect(len(err.Error())).To(BeNumerically("<", 300))
	g.Expect(err.Error()).NotTo(ContainSubstring("secret-key"))
}
