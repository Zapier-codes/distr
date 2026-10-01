ALTER TABLE TenantConfig
  DROP CONSTRAINT TenantConfig_release_complete,
  DROP COLUMN release_repository,
  DROP COLUMN release_id,
  DROP COLUMN release_asset_id;
