-- One row per device that has used its free product (f.xii). A device is only ever known by the salted hash of
-- the fingerprint its browser computed, never by the fingerprint itself, so the table cannot be used to recognize
-- a visitor without the server's salt. The row points at the tenant record the free product became, which keeps
-- one claim per tenant and lets an operator trace a claim back.
CREATE TABLE FreeTierClaim (
  id               UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at       TIMESTAMP NOT NULL DEFAULT now(),
  fingerprint_hash BYTEA     NOT NULL UNIQUE CHECK (octet_length(fingerprint_hash) = 32),
  tenant_config_id UUID      NOT NULL UNIQUE REFERENCES TenantConfig(id)
);
