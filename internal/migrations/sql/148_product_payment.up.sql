-- One-time price of a storefront product (f.xiii), in the minor unit of its currency (kobo, cents). Both are NULL
-- until the operator sets a price, and a product without a price cannot be paid for, so a request for it that the
-- free tier did not cover stays awaiting_gate.
ALTER TABLE ProductService
  ADD COLUMN price_minor    INT  CHECK (price_minor > 0),
  ADD COLUMN price_currency TEXT CHECK (price_currency ~ '^[A-Z]{3}$'),
  ADD CONSTRAINT ProductService_price_complete CHECK ((price_minor IS NULL) = (price_currency IS NULL));

-- The charge made for one tenant record through B-Pay. The amount and currency are copied here when the payment is
-- created, so a later price change never alters what a payment in flight has to cover. A row with no paid_at is a
-- payment link that has not been paid yet.
CREATE TABLE ProductPayment (
  id               UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at       TIMESTAMP NOT NULL DEFAULT now(),
  tenant_config_id UUID      NOT NULL UNIQUE REFERENCES TenantConfig(id),
  amount_minor     INT       NOT NULL CHECK (amount_minor > 0),
  currency         TEXT      NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  bpay_payment_id  TEXT      NOT NULL UNIQUE CHECK (bpay_payment_id <> ''),
  payment_link     TEXT      NOT NULL CHECK (payment_link ~ '^https?://'),
  paid_at          TIMESTAMP
);
