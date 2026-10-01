package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/bpay"
	"github.com/distr-sh/distr/internal/buildtrigger"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/devicefingerprint"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/productpayment"
	"github.com/distr-sh/distr/internal/tenantconfig"
	"github.com/distr-sh/distr/internal/turnstile"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
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
	r.Post("/{id}/payment", getProductRequestPaymentHandler).
		With(option.Description("Get the payment page of a request that has to be paid for, creating the payment if " +
			"it does not exist yet. The request id is the reference given when it was submitted")).
		With(option.Response(http.StatusOK, api.ProductRequestPayment{}))
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
	var service *types.ProductService
	gate := types.ProductRequestGatePaymentRequired
	err = db.RunTx(ctx, func(ctx context.Context) error {
		var err error
		if service, err = db.GetActiveProductServiceByID(ctx, request.ProductServiceID); err != nil {
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

		// The claim is made in the same transaction as the records it is for, so a request that fails after it
		// does not use up the free product of the device.
		gate, tenant, err = applyFreeTierGate(ctx, request.DeviceFingerprint, tenant)
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

	var paymentURL *string
	if gate == types.ProductRequestGateFree && service.Type == types.ProductServiceTypeApp {
		createdTenant = dispatchFreeBuild(ctx, createdTenant)
	} else if gate == types.ProductRequestGatePaymentRequired {
		// The request is registered either way. When no payment page can be made now, the requester asks for it again
		// through the payment endpoint, or the operator sees an awaiting_gate record.
		if link, err := productpayment.EnsurePaymentLink(ctx, createdTenant, service); err == nil {
			paymentURL = &link
		} else if !errors.Is(err, productpayment.ErrNoPrice) && !errors.Is(err, bpay.ErrNotConfigured) {
			log.Warn("could not create the payment of a product request",
				zap.String("tenantId", createdTenant.TenantID), zap.Error(err))
		}
	}

	RespondJSONWithStatus(w, http.StatusCreated,
		mapping.ProductRequestToAPI(*createdRequest, *createdTenant, gate, paymentURL))
}

func getProductRequestPaymentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "that request does not exist", http.StatusNotFound)
		return
	}
	request, err := db.GetProductRequest(ctx, id)
	if err == nil && request.TenantConfigID == nil {
		err = apierrors.ErrNotFound
	}
	var tenant *types.TenantConfig
	var service *types.ProductService
	if err == nil {
		tenant, _, err = db.GetTenantConfigForBuild(ctx, *request.TenantConfigID)
	}
	if err == nil {
		service, err = db.GetActiveProductServiceByID(ctx, tenant.ProductServiceID)
	}
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Error(w, "that request does not exist", http.StatusNotFound)
		return
	} else if err != nil {
		log.Warn("could not load a product request for payment", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	link, err := productpayment.EnsurePaymentLink(ctx, tenant, service)
	switch {
	case err == nil:
		RespondJSON(w, api.ProductRequestPayment{PaymentURL: link})
	case errors.Is(err, productpayment.ErrNotAwaitingPayment):
		http.Error(w, "this request does not need a payment, or has been paid already", http.StatusConflict)
	case errors.Is(err, productpayment.ErrNoPrice), errors.Is(err, bpay.ErrNotConfigured):
		http.Error(w, "paying for this product is not possible yet, we will contact you", http.StatusConflict)
	default:
		log.Warn("could not create the payment of a product request",
			zap.String("tenantId", tenant.TenantID), zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, "the payment service could not be reached, please try again in a moment",
			http.StatusBadGateway)
	}
}

// applyFreeTierGate is the free/paid check (f.xii). A request is free when the instance has a salt, the browser sent
// a fingerprint, and no earlier request from that device claimed its free product; the record then moves from
// awaiting_gate to queued, which is the only way a record reaches a build. Every other request is left
// awaiting_gate for the payment step (f.xiii). The fingerprint is only ever stored as a salted hash.
func applyFreeTierGate(
	ctx context.Context, deviceFingerprint string, tenant *types.TenantConfig,
) (types.ProductRequestGate, *types.TenantConfig, error) {
	salt := env.DeviceFingerprintSalt()
	if salt == nil || deviceFingerprint == "" {
		return types.ProductRequestGatePaymentRequired, tenant, nil
	}

	hash, err := devicefingerprint.Hash(*salt, deviceFingerprint)
	if err != nil {
		return "", nil, err
	}
	claimed, err := db.ClaimFreeTier(ctx, hash, tenant.ID)
	if err != nil {
		return "", nil, err
	}
	if !claimed {
		return types.ProductRequestGatePaymentRequired, tenant, nil
	}

	queued, err := db.TransitionTenantBuildStatus(
		ctx, tenant.ID, types.TenantBuildStatusAwaitingGate, types.TenantBuildStatusQueued, nil)
	if err != nil {
		return "", nil, err
	}
	return types.ProductRequestGateFree, queued, nil
}

// dispatchFreeBuild starts the build of a record the gate has queued and returns the record as it is afterwards. A
// failure to dispatch is not the requester's: the request is registered either way, and the record is left queued
// (nothing configured yet) or failed (GitHub refused), both visible to the operator.
func dispatchFreeBuild(ctx context.Context, tenant *types.TenantConfig) *types.TenantConfig {
	log := internalctx.GetLogger(ctx)
	// The request is registered, so the dispatch must not be cut short because the requester went away.
	ctx = context.WithoutCancel(ctx)

	if err := buildtrigger.DispatchTenantBuild(ctx, tenant.ID); errors.Is(err, buildtrigger.ErrNotConfigured) {
		log.Warn("a free tenant build is queued but dispatch is not configured",
			zap.String("tenantId", tenant.TenantID))
	} else if err != nil {
		log.Warn("could not dispatch a free tenant build",
			zap.String("tenantId", tenant.TenantID), zap.Error(err))
	}

	current, _, err := db.GetTenantConfigForBuild(ctx, tenant.ID)
	if err != nil {
		log.Warn("could not reload a tenant record after dispatch", zap.Error(err))
		return tenant
	}
	return current
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
