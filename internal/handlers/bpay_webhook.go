package handlers

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/bpay"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/productpayment"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

const maxBPayWebhookBodyBytes = 64 * 1024

// PublicBPayWebhookRouter receives B-Pay's outgoing webhooks (leaf f.xiii). A webhook is only believed when its
// signature matches, and even then the payment is retrieved from B-Pay before anything is queued.
func PublicBPayWebhookRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Product Requests"))
	r.Use(httprate.LimitBy(120, 1*time.Minute, productRequestClientIPKey))
	r.Post("/", postBPayWebhookHandler).
		With(option.Hidden(true)).
		With(option.Description("Receive a signed B-Pay webhook about a payment of a product request")).
		With(option.Response(http.StatusOK, nil))
}

func postBPayWebhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	config := env.BPay()
	if config == nil {
		// Not configured, so no payment can be in flight. 404 does not tell a scanner that the endpoint exists.
		http.NotFound(w, r)
		return
	}

	// The signature is over the exact bytes, so the body is read whole, bounded, before anything parses it.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBPayWebhookBodyBytes))
	if err != nil {
		http.Error(w, "the body is missing or too large", http.StatusRequestEntityTooLarge)
		return
	}
	if !bpay.VerifySignature(config.WebhookSecret, body, r.Header.Get(bpay.SignatureHeader)) {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	event, err := bpay.ParseWebhookEvent(body)
	if err != nil {
		http.Error(w, "the webhook is malformed", http.StatusBadRequest)
		return
	}
	// Only a success changes anything. Every other event is acknowledged so that B-Pay does not resend it.
	if event.EventType != bpay.EventPaymentSucceeded || event.Content.Object.PaymentID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = productpayment.ConfirmPaid(ctx, event.Content.Object.PaymentID)
	switch {
	case err == nil,
		errors.Is(err, productpayment.ErrUnknownPayment),
		errors.Is(err, productpayment.ErrNotSucceeded):
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, productpayment.ErrMismatch):
		// Resending cannot make the payment cover more, so this is acknowledged, and left for the operator.
		log.Error("a B-Pay payment does not match the amount that was asked for",
			zap.String("paymentId", event.Content.Object.PaymentID), zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusOK)
	default:
		// B-Pay unreachable or a database error: not acknowledged, so B-Pay sends the webhook again.
		log.Warn("could not confirm a B-Pay payment",
			zap.String("paymentId", event.Content.Object.PaymentID), zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
