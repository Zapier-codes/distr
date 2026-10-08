package handlers

import (
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

// g.ii (D5): a listing price is USD cents. Both halves are required together, the amount is positive, and any
// other currency is refused.
func TestNormalizeListingPrice(t *testing.T) {
	g := NewWithT(t)

	amount := 1500
	usd := "USD"
	eur := "eur"

	gotAmount, gotCurrency, err := normalizeListingPrice(&amount, &usd)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*gotAmount).To(Equal(1500))
	g.Expect(*gotCurrency).To(Equal("USD"))

	// Case-insensitive on the way in, always stored upper-case.
	_, gotCurrency, err = normalizeListingPrice(&amount, &eur)
	g.Expect(err).To(HaveOccurred())

	// Both nil means "no price", which is allowed.
	gotAmount, gotCurrency, err = normalizeListingPrice(nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gotAmount).To(BeNil())
	g.Expect(gotCurrency).To(BeNil())

	// Half a price is refused.
	_, _, err = normalizeListingPrice(&amount, nil)
	g.Expect(err).To(HaveOccurred())
	_, _, err = normalizeListingPrice(nil, &usd)
	g.Expect(err).To(HaveOccurred())

	// Zero and negative are refused.
	zero := 0
	_, _, err = normalizeListingPrice(&zero, &usd)
	g.Expect(err).To(HaveOccurred())
	negative := -1
	_, _, err = normalizeListingPrice(&negative, &usd)
	g.Expect(err).To(HaveOccurred())
}

func TestListingSlug(t *testing.T) {
	g := NewWithT(t)

	slug, err := listingSlug("  My Great App!! ")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(slug).To(Equal("my-great-app"))

	slug, err = listingSlug("Ünïcödé  Åpp")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(slug).To(Equal("n-c-d-pp"))

	_, err = listingSlug("!!!")
	g.Expect(err).To(HaveOccurred())
	_, err = listingSlug("")
	g.Expect(err).To(HaveOccurred())
}

// A new listing always starts as a draft owned by the caller, never live, whatever the body asked for.
func TestCreateListingFromRequest(t *testing.T) {
	g := NewWithT(t)

	owner := uuid.New()
	amount := 999
	usd := "USD"
	listing, err := createListingFromRequest(api.CreateProductListingRequest{
		Type:             types.ProductServiceTypeApp,
		Name:             "Neat Tool",
		Summary:          "Does one thing well",
		PriceAmountMinor: &amount,
		PriceCurrency:    &usd,
	}, owner)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(listing.ListingStatus).To(Equal(types.ProductServiceListingStatusDraft))
	g.Expect(listing.Active).To(BeFalse())
	g.Expect(listing.Slug).To(Equal("neat-tool"))
	g.Expect(*listing.OwnerUserAccountID).To(Equal(owner))
	g.Expect(*listing.PriceMinor).To(Equal(999))
	g.Expect(*listing.PriceCurrency).To(Equal("USD"))

	_, err = createListingFromRequest(api.CreateProductListingRequest{Name: "!!!"}, owner)
	g.Expect(err).To(HaveOccurred())
	_, err = createListingFromRequest(api.CreateProductListingRequest{Name: "ok", Type: "nonsense"}, owner)
	g.Expect(err).To(HaveOccurred())
}

// Publishing (status live) turns the listing on and requires it to be presentable; every other status keeps it off.
func TestApplyListingUpdate(t *testing.T) {
	g := NewWithT(t)

	listing := types.ProductService{
		Name:          "Neat Tool",
		Summary:       "Does one thing well",
		ListingStatus: types.ProductServiceListingStatusDraft,
	}

	live := types.ProductServiceListingStatusLive
	g.Expect(applyListingUpdate(&listing, api.UpdateProductListingRequest{ListingStatus: &live})).
		To(Succeed())
	g.Expect(listing.Active).To(BeTrue())
	g.Expect(listing.ListingStatus).To(Equal(live))

	// Going live without a summary is refused.
	bare := types.ProductService{Name: "No summary", ListingStatus: types.ProductServiceListingStatusDraft}
	g.Expect(applyListingUpdate(&bare, api.UpdateProductListingRequest{ListingStatus: &live})).
		To(HaveOccurred())

	// An unknown status is refused.
	weird := types.ProductServiceListingStatus("manual_review")
	g.Expect(applyListingUpdate(&listing, api.UpdateProductListingRequest{ListingStatus: &weird})).
		To(HaveOccurred())

	// Suspending a live listing takes it off the storefront.
	suspended := types.ProductServiceListingStatusSuspended
	g.Expect(applyListingUpdate(&listing, api.UpdateProductListingRequest{ListingStatus: &suspended})).
		To(Succeed())
	g.Expect(listing.Active).To(BeFalse())

	// Clearing the price returns the listing to free.
	g.Expect(applyListingUpdate(&listing, api.UpdateProductListingRequest{ClearPrice: true})).To(Succeed())
	g.Expect(listing.PriceMinor).To(BeNil())
	g.Expect(listing.PriceCurrency).To(BeNil())

	// A non-USD price is refused on update too.
	amount := 500
	eur := "EUR"
	g.Expect(applyListingUpdate(&listing, api.UpdateProductListingRequest{
		PriceAmountMinor: &amount, PriceCurrency: &eur,
	})).To(HaveOccurred())
}
