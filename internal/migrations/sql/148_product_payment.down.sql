DROP TABLE ProductPayment;
ALTER TABLE ProductService
  DROP CONSTRAINT ProductService_price_complete,
  DROP COLUMN price_minor,
  DROP COLUMN price_currency;
