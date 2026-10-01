-- The download link of a finished build (f.v). The mail carries a random token that opens a distr endpoint, which
-- resolves the GitHub Release asset at click time. Only the SHA-256 of the token is stored, so the database cannot
-- be used to build a link.
ALTER TABLE TenantConfig
  ADD COLUMN download_token_hash       BYTEA     UNIQUE CHECK (octet_length(download_token_hash) = 32),
  ADD COLUMN download_token_expires_at TIMESTAMP,
  -- Set when Novu accepted the mail. A report that is repeated sends nothing once it is set.
  ADD COLUMN build_email_sent_at       TIMESTAMP,
  ADD CONSTRAINT TenantConfig_download_token_complete CHECK (
    (download_token_hash IS NULL) = (download_token_expires_at IS NULL)
  );
