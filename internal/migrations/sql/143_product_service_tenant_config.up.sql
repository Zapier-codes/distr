CREATE TYPE PRODUCT_SERVICE_TYPE AS ENUM ('app', 'website');

-- ProductService is one entry of the public request storefront: something a visitor can browse and request.
CREATE TABLE ProductService (
  id          UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at  TIMESTAMP            NOT NULL DEFAULT now(),
  slug        TEXT                 NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$'),
  type        PRODUCT_SERVICE_TYPE NOT NULL,
  name        TEXT                 NOT NULL,
  summary     TEXT                 NOT NULL,
  description TEXT                 NOT NULL DEFAULT '',
  active      BOOLEAN              NOT NULL DEFAULT true,
  sort_order  INT                  NOT NULL DEFAULT 0
);

-- Demo material shown on the storefront. A URL, never an upload: Distr stores no media.
CREATE TABLE ProductServiceMedia (
  id                 UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
  product_service_id UUID      NOT NULL REFERENCES ProductService(id) ON DELETE CASCADE,
  kind               TEXT      NOT NULL CHECK (kind IN ('screenshot', 'video')),
  url                TEXT      NOT NULL CHECK (url ~ '^https://'),
  caption            TEXT,
  sort_order         INT       NOT NULL DEFAULT 0
);

CREATE INDEX ProductServiceMedia_product_service_id ON ProductServiceMedia(product_service_id);

CREATE TYPE TENANT_BUILD_STATUS AS ENUM (
  -- Created by the request flow. Nothing is dispatched until the free/paid gate (f.xii, f.xiii) has cleared it.
  'awaiting_gate',
  'queued',
  'building',
  'succeeded',
  'failed'
);

-- TenantConfig is the canonical tenant record in the shape of Storeapp's spec/tenant-config-schema.md (v1).
-- generated_at and expires_at are not stored: they belong to a publish, not to the record.
-- It holds nothing about the requester. The contact email stays in ProductRequest, because this record is
-- what gets signed and published to other services.
CREATE TABLE TenantConfig (
  id                      UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at              TIMESTAMP           NOT NULL DEFAULT now(),
  tenant_id               TEXT                NOT NULL UNIQUE
    CHECK (tenant_id ~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$'),
  product_service_id      UUID                NOT NULL REFERENCES ProductService(id),
  display_name            TEXT                NOT NULL CHECK (display_name <> ''),
  primary_color_hex       TEXT                NOT NULL CHECK (primary_color_hex ~ '^#[0-9a-fA-F]{6}$'),
  logo                    BYTEA,
  logo_content_type       TEXT,
  logo_sha256             TEXT                CHECK (logo_sha256 ~ '^[a-f0-9]{64}$'),
  cdn_base                TEXT                NOT NULL DEFAULT 'https://nikhilkain.github.io/appstore-metadata',
  catalog_index_base_url  TEXT,
  domains                 TEXT[]              NOT NULL DEFAULT '{}',
  -- 0 until the first publish. Strictly increasing per publish after that (anti-rollback).
  sequence                INT                 NOT NULL DEFAULT 0 CHECK (sequence >= 0),
  build_status            TENANT_BUILD_STATUS NOT NULL DEFAULT 'awaiting_gate',
  build_status_updated_at TIMESTAMP           NOT NULL DEFAULT now(),
  CONSTRAINT TenantConfig_logo_complete CHECK (
    (logo IS NULL) = (logo_content_type IS NULL) AND (logo IS NULL) = (logo_sha256 IS NULL)
  )
);

ALTER TABLE ProductRequest
  ADD COLUMN tenant_config_id UUID UNIQUE REFERENCES TenantConfig(id);

-- The two storefront entries every instance starts with. Their demo media is added by the operator.
INSERT INTO ProductService (slug, type, name, summary, description, sort_order) VALUES
  ('branded-app-store', 'app', 'Branded app store',
   'Your own Android app store, with your name, your colors and your icon.',
   'A white-label build of the app store for Android. You choose the name, the theme color and the icon, and ' ||
   'receive a signed APK by email once it is built.',
   10),
  ('branded-web-store', 'website', 'Branded web store',
   'Your own storefront website, with your name and your colors.',
   'A white-label storefront website. You choose the name and the theme color, and the site is provisioned for ' ||
   'you without a separate build.',
   20);
