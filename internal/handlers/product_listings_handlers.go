package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

// DeveloperListingsRouter is the developer's own listings (g.iii-b). It is mounted under the authenticated vendor
// group, so it is reachable only with a logged-in account; the handlers then narrow every operation to the listings
// the caller owns. Developers, the very role the vendor group refuses elsewhere, are the users this router is for,
// so it is mounted in a group of its own that does not carry RequireNonDeveloper.
func DeveloperListingsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Developer Listings"))
	r.Get("/", listDeveloperListingsHandler).
		With(option.Description("List the listings owned by the signed-in developer, drafts included")).
		With(option.Response(http.StatusOK, []api.DeveloperListing{}))
	r.Post("/", createDeveloperListingHandler).
		With(option.Description("Create a listing. It starts as a draft and is not shown to visitors until published")).
		With(option.Request(api.CreateProductListingRequest{})).
		With(option.Response(http.StatusCreated, api.DeveloperListing{}))
	r.Get("/{id}", getDeveloperListingHandler).
		With(option.Description("Get one of the signed-in developer's listings")).
		With(option.Response(http.StatusOK, api.DeveloperListing{}))
	r.Patch("/{id}", updateDeveloperListingHandler).
		With(option.Description("Edit a listing the signed-in developer owns. Publishing makes it purchasable")).
		With(option.Request(api.UpdateProductListingRequest{})).
		With(option.Response(http.StatusOK, api.DeveloperListing{}))
}

func currentUserID(r *http.Request) uuid.UUID {
	return auth.Authentication.Require(r.Context()).CurrentUserID()
}

func listDeveloperListingsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	listings, err := db.ListProductListingsByOwner(ctx, currentUserID(r))
	if err != nil {
		log.Warn("could not list developer listings", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSON(w, mapping.List(listings, mapping.DeveloperListingToAPI))
}

func createDeveloperListingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	request, err := JsonBody[api.CreateProductListingRequest](w, r)
	if err != nil {
		return
	}

	listing, err := createListingFromRequest(request, currentUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := db.CreateProductListing(ctx, listing); err != nil {
		if errors.Is(err, apierrors.ErrAlreadyExists) {
			http.Error(w, "a listing with a similar name already exists, please choose another name",
				http.StatusConflict)
			return
		}
		log.Warn("could not create developer listing", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSONWithStatus(w, http.StatusCreated, mapping.DeveloperListingToAPI(*listing))
}

func getDeveloperListingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "that listing does not exist", http.StatusNotFound)
		return
	}
	listing, err := db.GetProductListingByOwnerAndID(ctx, currentUserID(r), id)
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Error(w, "that listing does not exist", http.StatusNotFound)
		return
	} else if err != nil {
		log.Warn("could not get developer listing", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSON(w, mapping.DeveloperListingToAPI(*listing))
}

func updateDeveloperListingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "that listing does not exist", http.StatusNotFound)
		return
	}
	request, err := JsonBody[api.UpdateProductListingRequest](w, r)
	if err != nil {
		return
	}

	listing, err := db.GetProductListingByOwnerAndID(ctx, currentUserID(r), id)
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Error(w, "that listing does not exist", http.StatusNotFound)
		return
	} else if err != nil {
		log.Warn("could not load developer listing", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := applyListingUpdate(listing, request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updated, err := db.UpdateProductListing(ctx, *listing)
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Error(w, "that listing does not exist", http.StatusNotFound)
		return
	} else if err != nil {
		log.Warn("could not update developer listing", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSON(w, mapping.DeveloperListingToAPI(*updated))
}
