ALTER TABLE ProductService
  DROP CONSTRAINT ProductService_price_currency_usd,
  DROP COLUMN listing_status,
  DROP COLUMN owner_user_account_id;
