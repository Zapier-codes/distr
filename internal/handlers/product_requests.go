package handlers

import (
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/turnstile"
	"github.com/getsentry/sentry-go"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func PublicProductRequestsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Product Requests"))
	r.Use(
		httprate.LimitBy(5, 1*time.Minute, productRequestClientIPKey),
		httprate.LimitBy(20, 1*time.Hour, productRequestClientIPKey),
	)
	r.Post("/", createProductRequestHandler).
		With(option.Description("Submit a request for a product build without an account. "+
			"No user, organization or customer organization is created or required")).
		With(option.Request(api.CreateProductRequestRequest{})).
		With(option.Response(http.StatusCreated, api.ProductRequest{}))
}

func productRequestClientIPKey(r *http.Request) (string, error) {
	return chimiddleware.GetClientIP(r.Context()), nil
}

func createProductRequestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	request, err := JsonBody[api.CreateProductRequestRequest](w, r)
	if err != nil {
		return
	}
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if env.TurnstileSiteKey() != nil {
		if err := turnstile.Verify(ctx, request.TurnstileToken, chimiddleware.GetClientIP(ctx)); err != nil {
			log.Info("turnstile verification failed", zap.Error(err))
			http.Error(w, "could not verify that you are human, please reload the page and try again",
				http.StatusForbidden)
			return
		}
	}

	created, err := db.CreateProductRequest(ctx, mapping.ProductRequestToInternal(request))
	if err != nil {
		log.Warn("could not create product request", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSONWithStatus(w, http.StatusCreated, mapping.ProductRequestToAPI(*created))
}
