-- g.ii (D5) and g.iii-b (D23, D24): developer listings.
--
-- A listing is a ProductService a developer owns. The two entries seeded by 143 and anything the operator added
-- have no owner, so they keep behaving exactly as before: the storefront's own catalogue, live by default.
ALTER TABLE ProductService
  -- The developer who owns the listing (g.iii-b). NULL for the storefront's own entries (the operator's).
  ADD COLUMN owner_user_account_id UUID REFERENCES UserAccount(id) ON DELETE SET NULL,
  -- The automatic-publish state (D23): 'draft' until the owner publishes, 'live' when it can be requested,
  -- 'suspended' when the operator takes it down. Default 'live' so the storefront's own entries are unaffected.
  ADD COLUMN listing_status TEXT NOT NULL DEFAULT 'live'
    CHECK (listing_status IN ('draft', 'live', 'suspended'));

CREATE INDEX ProductService_owner_user_account_id ON ProductService(owner_user_account_id);

-- g.ii (D5): developer listing prices are USD cents only. 148 allowed any ISO 4217 code; constrain what is stored
-- and what may be written from here on. The storefront's own entries with another currency are corrected to USD.
UPDATE ProductService SET price_currency = 'USD' WHERE price_currency IS NOT NULL AND price_currency <> 'USD';
ALTER TABLE ProductService
  ADD CONSTRAINT ProductService_price_currency_usd CHECK (price_currency IS NULL OR price_currency = 'USD');
