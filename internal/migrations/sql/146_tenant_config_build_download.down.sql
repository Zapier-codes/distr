ALTER TABLE TenantConfig
  DROP CONSTRAINT TenantConfig_download_token_complete,
  DROP COLUMN download_token_hash,
  DROP COLUMN download_token_expires_at,
  DROP COLUMN build_email_sent_at;
