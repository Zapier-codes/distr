package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/buildnotify"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

const (
	maxBuildStatusBodyBytes = 16 * 1024
	defaultFailedMessage    = "the build failed, the workflow gave no reason"
)

// PublicBuildStatusRouter receives the end of a tenant build from the workflow of Storeapp (leaf f.iv.zo). It is
// behind the same bearer token as the build-config endpoint.
func PublicBuildStatusRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Product Requests"))
	r.Use(
		middleware.RequestSize(maxBuildStatusBodyBytes),
		httprate.LimitBy(30, 1*time.Minute, productRequestClientIPKey),
	)
	r.Post("/", postBuildStatusHandler).
		With(option.Hidden(true)).
		With(option.Description("Report that the build of a tenant record succeeded, with its GitHub Release asset, "+
			"or failed. Needs the build-config bearer token, and only applies to a record that is building. "+
			"Repeating a report that was already applied answers the same")).
		With(option.Request(api.BuildStatusRequest{})).
		With(option.Response(http.StatusNoContent, nil))
}

func postBuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	expected := env.StoreappBuildConfigToken()
	buildConfig := env.StoreappBuild()
	if expected == nil || buildConfig == nil {
		// Not configured, so no build can be running. 404 does not tell a scanner that the endpoint exists.
		http.NotFound(w, r)
		return
	}
	given, ok := bearerToken(r)
	if !ok || !bearerTokenMatches(given, *expected) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	request, err := JsonBody[api.BuildStatusRequest](w, r)
	if err != nil {
		return
	}
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, productType, err := db.GetTenantConfigForBuild(ctx, request.TenantConfigID)
	if errors.Is(err, apierrors.ErrNotFound) || (err == nil && productType != types.ProductServiceTypeApp) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		buildStatusInternalError(w, r, "could not get tenant config for build status", err)
		return
	}

	// distr will later fetch this asset with its own GitHub token (f.v), so the token holder must not be able to
	// point that at another repository than the one the builds are dispatched to.
	if request.Status == types.TenantBuildStatusSucceeded &&
		!strings.EqualFold(request.Asset.Repository, buildConfig.Repository) {
		http.Error(w, "asset.repository is not the repository the builds run in", http.StatusBadRequest)
		return
	}

	var message *string
	var assetRepository string
	var releaseID, assetID int64
	if request.Status == types.TenantBuildStatusFailed {
		message = new(defaultFailedMessage)
		if request.Message != nil && *request.Message != "" {
			message = request.Message
		}
	} else {
		assetRepository, releaseID, assetID = request.Asset.Repository, request.Asset.ReleaseID, request.Asset.AssetID
	}

	_, err = db.CompleteTenantBuild(
		ctx, request.TenantConfigID, request.Status, message, assetRepository, releaseID, assetID)
	if errors.Is(err, apierrors.ErrConflict) {
		// Either the record is not building, or this report was already applied and the workflow is retrying.
		current, _, getErr := db.GetTenantConfigForBuild(ctx, request.TenantConfigID)
		if getErr != nil {
			buildStatusInternalError(w, r, "could not get tenant config for build status", getErr)
			return
		}
		if buildReportAlreadyApplied(request, *current) {
			// A repeat is how a mail that Novu did not accept gets another attempt.
			if mailBuildReady(w, r, request) {
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "the build of this record is not running", http.StatusConflict)
		return
	} else if err != nil {
		buildStatusInternalError(w, r, "could not complete tenant build", err)
		return
	}

	log.Info("tenant build reported",
		zap.String("tenantConfigId", request.TenantConfigID.String()), zap.String("status", string(request.Status)))
	if mailBuildReady(w, r, request) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// mailBuildReady hands the mail about a succeeded build off to Novu, and reports whether it answered the request
// with an error. The build is already recorded by then, so the error says so, and the workflow's repeat of the report
// is what tries again. A failed build sends nothing, and an instance without Novu records the build and sends nothing.
func mailBuildReady(w http.ResponseWriter, r *http.Request, request api.BuildStatusRequest) bool {
	if request.Status != types.TenantBuildStatusSucceeded {
		return false
	}
	ctx := r.Context()
	err := buildnotify.SendBuildReady(ctx, request.TenantConfigID, publicBaseURL(env.Host(), env.HostScheme()))
	if errors.Is(err, buildnotify.ErrNotConfigured) {
		internalctx.GetLogger(ctx).Warn("a tenant build succeeded but NOVU_API_KEY is not set, no mail was sent",
			zap.String("tenantConfigId", request.TenantConfigID.String()))
		return false
	} else if err != nil {
		internalctx.GetLogger(ctx).Warn("could not hand the build mail off", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, "the build was recorded, but the mail could not be sent, report it again to retry",
			http.StatusBadGateway)
		return true
	}
	return false
}

// buildReportAlreadyApplied reports whether current is the state the report would have produced. A failed report
// matches on the status alone, since its message is only for the operator. A succeeded one has to name the same asset,
// so that a second, different asset cannot be passed off as a retry.
func buildReportAlreadyApplied(request api.BuildStatusRequest, current types.TenantConfig) bool {
	if current.BuildStatus != request.Status {
		return false
	}
	if request.Status == types.TenantBuildStatusFailed {
		return true
	}
	return current.ReleaseRepository != nil && current.ReleaseID != nil && current.ReleaseAssetID != nil &&
		*current.ReleaseRepository == request.Asset.Repository &&
		*current.ReleaseID == request.Asset.ReleaseID &&
		*current.ReleaseAssetID == request.Asset.AssetID
}

func buildStatusInternalError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	ctx := r.Context()
	internalctx.GetLogger(ctx).Warn(msg, zap.Error(err))
	sentry.GetHubFromContext(ctx).CaptureException(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
