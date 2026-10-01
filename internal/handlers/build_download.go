package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/buildnotify"
	"github.com/distr-sh/distr/internal/buildtrigger"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

// PublicBuildDownloadsRouter is where the button of the "your app is ready" mail points (leaf f.v). The token in the
// path is the whole credential. It is not for the workflow, and it needs no bearer token.
func PublicBuildDownloadsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Product Requests"))
	r.Use(httprate.LimitBy(30, 1*time.Minute, productRequestClientIPKey))
	r.Get("/{token}", getBuildDownloadHandler).
		With(option.Hidden(true)).
		With(option.Description("Redirect to the short-lived signed download URL of a built app, which is resolved " +
			"server-side at click time"))
}

func getBuildDownloadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	// A mail scanner or a person who mistyped the link ends up here, and a link that is not even shaped like a token
	// does not reach the database.
	token := r.PathValue("token")
	if len(token) != buildnotify.TokenLength {
		http.NotFound(w, r)
		return
	}

	tenant, err := db.GetTenantConfigByDownloadToken(ctx, buildnotify.HashToken(token))
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Error(w, "this download link is not valid, or has expired", http.StatusNotFound)
		return
	} else if err != nil {
		buildStatusInternalError(w, r, "could not get tenant config by download token", err)
		return
	}

	buildConfig := env.StoreappBuild()
	if buildConfig == nil {
		http.Error(w, "downloads are not available right now, please try again later", http.StatusServiceUnavailable)
		return
	}
	// The record was checked against the configured repository when the build was reported. This is the call that
	// sends distr's GitHub token, so it is checked again.
	if tenant.ReleaseRepository == nil || tenant.ReleaseAssetID == nil ||
		!strings.EqualFold(*tenant.ReleaseRepository, buildConfig.Repository) {
		log.Warn("download refused, the release asset is not in the configured repository",
			zap.String("tenantConfigId", tenant.ID.String()))
		http.NotFound(w, r)
		return
	}

	location, err := buildtrigger.NewClient(*buildConfig, "").ResolveAssetDownload(
		ctx, *tenant.ReleaseRepository, *tenant.ReleaseAssetID)
	if errors.Is(err, buildtrigger.ErrAssetNotFound) {
		log.Warn("download failed, GitHub does not have the asset", zap.String("tenantConfigId", tenant.ID.String()))
		http.Error(w, "this file is no longer available", http.StatusNotFound)
		return
	} else if err != nil {
		log.Warn("could not resolve the release asset", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, "the download could not be prepared, please try again in a moment", http.StatusBadGateway)
		return
	}

	// The signed URL is short-lived and belongs to this one request: nothing may keep it, and the page it leads to
	// must not learn which link led there.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, location, http.StatusFound)
}
