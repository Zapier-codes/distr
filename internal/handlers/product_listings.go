package handlers

import (
	"fmt"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

// normalizeListingPrice validates a developer-supplied price (g.ii, D5). The only currency a listing may be priced
// in is USD, and the amount is in USD cents, a positive integer. Both fields must be given together or not at all;
// an amount without a currency, or a currency without an amount, is refused. A nil amount and nil currency means
// "no price", which is allowed: a listing can be free. The returned pair is what should be stored.
func normalizeListingPrice(amountMinor *int, currency *string) (*int, *string, error) {
	if amountMinor == nil && currency == nil {
		return nil, nil, nil
	}
	if amountMinor == nil || currency == nil {
		return nil, nil, validation.NewValidationFailedError("a price needs both an amount and a currency")
	}
	if *amountMinor <= 0 {
		return nil, nil, validation.NewValidationFailedError("the price must be greater than zero")
	}
	if !strings.EqualFold(strings.TrimSpace(*currency), types.PriceCurrencyUSD) {
		return nil, nil, validation.NewValidationFailedError(
			fmt.Sprintf("listings are priced in %v only", types.PriceCurrencyUSD))
	}
	usd := types.PriceCurrencyUSD
	return amountMinor, &usd, nil
}

// listingSlug derives a URL slug from a listing name. It lowercases, keeps [a-z0-9], turns runs of anything else
// into a single hyphen, and trims hyphens. An empty result is an error: a listing must have a usable slug, and the
// database's slug CHECK enforces the same shape.
func listingSlug(name string) (string, error) {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "", validation.NewValidationFailedError("the name must contain at least one letter or digit")
	}
	if len(slug) > 63 {
		slug = strings.Trim(slug[:63], "-")
	}
	return slug, nil
}

// validateListingStatus accepts only the automatic-publish states (D23). The manual-review states of D9 are not
// part of the model, so they are refused rather than silently stored.
func validateListingStatus(status types.ProductServiceListingStatus) error {
	switch status {
	case types.ProductServiceListingStatusDraft,
		types.ProductServiceListingStatusLive,
		types.ProductServiceListingStatusSuspended:
		return nil
	default:
		return validation.NewValidationFailedError("unknown listing status")
	}
}

// createListingFromRequest validates a create request and builds the listing to insert. The owner is set here from
// the session, never from the request.
func createListingFromRequest(request api.CreateProductListingRequest, ownerID uuid.UUID) (*types.ProductService, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return nil, validation.NewValidationFailedError("name is empty")
	}
	slug, err := listingSlug(name)
	if err != nil {
		return nil, err
	}
	if request.Type != types.ProductServiceTypeApp && request.Type != types.ProductServiceTypeWebsite {
		return nil, validation.NewValidationFailedError("type must be app or website")
	}
	amount, currency, err := normalizeListingPrice(request.PriceAmountMinor, request.PriceCurrency)
	if err != nil {
		return nil, err
	}
	return &types.ProductService{
		Slug:               slug,
		Type:               request.Type,
		Name:               name,
		Summary:            strings.TrimSpace(request.Summary),
		Description:        request.Description,
		Active:             false,
		OwnerUserAccountID: &ownerID,
		ListingStatus:      types.ProductServiceListingStatusDraft,
		PriceMinor:         amount,
		PriceCurrency:      currency,
	}, nil
}

// applyListingUpdate validates an update request and applies it to a listing the caller owns. A nil field is left
// unchanged. Publishing (status live) requires a non-empty name and summary, since a live listing is shown to
// visitors; a suspended or draft listing may be incomplete.
func applyListingUpdate(listing *types.ProductService, request api.UpdateProductListingRequest) error {
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			return validation.NewValidationFailedError("name cannot be empty")
		}
		listing.Name = name
	}
	if request.Summary != nil {
		listing.Summary = strings.TrimSpace(*request.Summary)
	}
	if request.Description != nil {
		listing.Description = *request.Description
	}
	if request.ListingStatus != nil {
		if err := validateListingStatus(*request.ListingStatus); err != nil {
			return err
		}
		listing.ListingStatus = *request.ListingStatus
	}
	switch {
	case request.ClearPrice:
		listing.PriceMinor, listing.PriceCurrency = nil, nil
	case request.PriceAmountMinor != nil || request.PriceCurrency != nil:
		amount, currency, err := normalizeListingPrice(request.PriceAmountMinor, request.PriceCurrency)
		if err != nil {
			return err
		}
		listing.PriceMinor, listing.PriceCurrency = amount, currency
	}

	// A listing becomes purchasable only when it is live, and a live listing has to be presentable.
	if listing.ListingStatus == types.ProductServiceListingStatusLive {
		if listing.Name == "" || listing.Summary == "" {
			return validation.NewValidationFailedError("a listing needs a name and a summary before it can go live")
		}
		listing.Active = true
	} else {
		listing.Active = false
	}
	return nil
}
