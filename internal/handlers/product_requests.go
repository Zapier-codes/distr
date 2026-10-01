package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/tenantconfig"
	"github.com/distr-sh/distr/internal/turnstile"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

// maxTenantIDAttempts bounds the retries when a generated tenant_id is already taken. A collision of the random
// suffix is rare enough that more than one is a bug, not bad luck.
const maxTenantIDAttempts = 5

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

// PublicProductServicesRouter serves the storefront catalog.
func PublicProductServicesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Product Requests"))
	r.Use(httprate.LimitBy(120, 1*time.Minute, productRequestClientIPKey))
	r.Get("/", listProductServicesHandler).
		With(option.Description("List the products that can be requested, with their demo screenshots and videos")).
		With(option.Response(http.StatusOK, []api.ProductService{}))
	r.Get("/{slug}", getProductServiceHandler).
		With(option.Description("Get one product that can be requested")).
		With(option.Response(http.StatusOK, api.ProductService{}))
}

// PublicTenantLogosRouter serves the icon a requester uploaded, which is what the logo_url of the tenant record
// points at.
func PublicTenantLogosRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Product Requests"))
	r.Use(httprate.LimitBy(120, 1*time.Minute, productRequestClientIPKey))
	r.Get("/{tenantId}", getTenantLogoHandler).
		With(option.Description("Get the icon of a tenant"))
}

func productRequestClientIPKey(r *http.Request) (string, error) {
	return chimiddleware.GetClientIP(r.Context()), nil
}

func createProductRequestHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	// The body carries the icon, so it is bounded before anything reads it. This endpoint has no account behind it.
	if r.ContentLength > api.MaxProductRequestBodyBytes {
		http.Error(w, "the request is too large, use a smaller icon", http.StatusRequestEntityTooLarge)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, api.MaxProductRequestBodyBytes)

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

	var createdRequest *types.ProductRequest
	var createdTenant *types.TenantConfig
	err = db.RunTx(ctx, func(ctx context.Context) error {
		if _, err := db.GetActiveProductServiceByID(ctx, request.ProductServiceID); err != nil {
			return err
		}

		tenant, err := createTenantConfig(ctx, mapping.ProductRequestToTenantConfig(request))
		if err != nil {
			return err
		}

		toCreate := mapping.ProductRequestToInternal(request)
		toCreate.TenantConfigID = &tenant.ID
		created, err := db.CreateProductRequest(ctx, toCreate)
		if err != nil {
			return err
		}

		createdRequest, createdTenant = created, tenant
		return nil
	})
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Error(w, "that product is not available, please choose another one", http.StatusBadRequest)
		return
	} else if err != nil {
		log.Warn("could not create product request", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSONWithStatus(w, http.StatusCreated, mapping.ProductRequestToAPI(*createdRequest, *createdTenant))
}

// createTenantConfig inserts the record under a freshly generated tenant_id, generating another when it is taken.
func createTenantConfig(ctx context.Context, tenant types.TenantConfig) (*types.TenantConfig, error) {
	for range maxTenantIDAttempts {
		tenantID, err := tenantconfig.GenerateTenantID(tenant.DisplayName)
		if err != nil {
			return nil, err
		}
		tenant.TenantID = tenantID
		created, err := db.CreateTenantConfig(ctx, tenant)
		if errors.Is(err, apierrors.ErrAlreadyExists) {
			continue
		}
		return created, err
	}
	return nil, errors.New("could not find an unused tenant_id")
}

func listProductServicesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	services, err := db.GetActiveProductServices(ctx)
	if err != nil {
		log.Warn("could not list product services", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSON(w, mapping.List(services, mapping.ProductServiceToAPI))
}

func getProductServiceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	service, err := db.GetActiveProductServiceBySlug(ctx, r.PathValue("slug"))
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Error(w, "that product does not exist", http.StatusNotFound)
		return
	} else if err != nil {
		log.Warn("could not get product service", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSON(w, mapping.ProductServiceToAPI(*service))
}

func getTenantLogoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	data, contentType, sha256, err := db.GetTenantLogo(ctx, r.PathValue("tenantId"))
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Warn("could not get tenant logo", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	etag := `"` + sha256 + `"`
	w.Header().Set("ETag", etag)
	// The bytes of a tenant's logo never change under the same tenant_id.
	w.Header().Set("Cache-Control", "public, max-age=3600")
	// The bytes are the requester's, so the browser must treat them as an image and nothing else.
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(data)
}
