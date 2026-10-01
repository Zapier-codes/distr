-- The GitHub Release asset a finished build produced, as the identifiers the build-status webhook (f.iv.zo) reports
-- and the download redirect of f.v resolves at click time. Never a URL: a private repository's asset cannot be
-- fetched from one, and no binary is stored here. All three are set together or none are.
ALTER TABLE TenantConfig
  ADD COLUMN release_repository TEXT   CHECK (release_repository ~ '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$'),
  ADD COLUMN release_id         BIGINT CHECK (release_id > 0),
  ADD COLUMN release_asset_id   BIGINT CHECK (release_asset_id > 0),
  ADD CONSTRAINT TenantConfig_release_complete CHECK (
    (release_repository IS NULL) = (release_id IS NULL) AND (release_repository IS NULL) = (release_asset_id IS NULL)
  );
