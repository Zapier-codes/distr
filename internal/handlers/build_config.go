package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

// PublicBuildConfigRouter serves the tenant record to the build workflow of Storeapp (leaf f.iv.zi). It is not for
// requesters: every call needs the bearer token of STOREAPP_BUILD_CONFIG_TOKEN.
func PublicBuildConfigRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Product Requests"))
	// The limit counts every call, not only the failed ones, so a guesser gets the same few attempts a minute as
	// everyone else. The workflow calls once per build.
	r.Use(httprate.LimitBy(30, 1*time.Minute, productRequestClientIPKey))
	r.Get("/{tenantConfigId}", getBuildConfigHandler).
		With(option.Hidden(true)).
		With(option.Description("Get the tenant record a build is running for. Needs the build-config bearer token, "+
			"and only answers while the build of the record is running")).
		With(option.Response(http.StatusOK, api.BuildConfig{}))
}

func getBuildConfigHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	expected := env.StoreappBuildConfigToken()
	if expected == nil {
		// Not configured. Answering 401 would tell a scanner that the endpoint exists, 404 does not.
		http.NotFound(w, r)
		return
	}
	given, ok := bearerToken(r)
	if !ok || !bearerTokenMatches(given, *expected) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(r.PathValue("tenantConfigId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	tenant, productType, err := db.GetTenantConfigForBuild(ctx, id)
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Warn("could not get tenant config for build", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// A website is never built, and a record the dispatch has not started has no business in a build. Neither is
	// told apart from an unknown id: the caller holds the token, but it only needs what its own build needs.
	if productType != types.ProductServiceTypeApp {
		http.NotFound(w, r)
		return
	}
	if tenant.BuildStatus != types.TenantBuildStatusBuilding {
		http.Error(w, "the build of this record is not running", http.StatusConflict)
		return
	}

	logoURL := publicBaseURL(env.Host(), env.HostScheme()) + "/api/public/v1/tenant-logos/" + tenant.TenantID

	// The answer carries a tenant's branding, so no cache between here and the workflow may keep it.
	w.Header().Set("Cache-Control", "no-store")
	RespondJSON(w, mapping.TenantConfigToBuildConfig(*tenant, logoURL, time.Now()))
}

// bearerToken returns the token of an "Authorization: Bearer <token>" header. The scheme is case-insensitive.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

// bearerTokenMatches compares in constant time. Both values are hashed first, so the comparison takes the same
// time whatever their lengths, and what it takes does not depend on how much of the token was right.
func bearerTokenMatches(given, expected string) bool {
	givenSum := sha256.Sum256([]byte(given))
	expectedSum := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(givenSum[:], expectedSum[:]) == 1
}

// publicBaseURL is the URL this instance is reached at. DISTR_HOST may or may not carry a scheme, and when it does,
// it wins, since it also carries a port that the scheme belongs to.
func publicBaseURL(host string, scheme env.URLScheme) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if strings.Contains(host, "://") {
		return host
	}
	return fmt.Sprintf("%v://%v", scheme, host)
}
