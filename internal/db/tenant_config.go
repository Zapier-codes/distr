package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/jackc/pgx/v5"
)

const tenantConfigOutputExpr = `
	tc.id, tc.created_at, tc.tenant_id, tc.product_service_id, tc.display_name, tc.primary_color_hex,
	tc.logo, tc.logo_content_type, tc.logo_sha256, tc.cdn_base, tc.catalog_index_base_url, tc.domains,
	tc.sequence, tc.build_status, tc.build_status_updated_at
`

// CreateTenantConfig inserts a record in the state awaiting_gate. It returns apierrors.ErrAlreadyExists when the
// tenant_id is taken, without aborting an enclosing transaction, so that the caller can retry with a new one.
func CreateTenantConfig(ctx context.Context, config types.TenantConfig) (*types.TenantConfig, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO TenantConfig AS tc (
			tenant_id, product_service_id, display_name, primary_color_hex, logo, logo_content_type, logo_sha256
		) VALUES (
			@tenantId, @productServiceId, @displayName, @primaryColorHex, @logo, @logoContentType, @logoSha256
		)
		ON CONFLICT (tenant_id) DO NOTHING
		RETURNING`+tenantConfigOutputExpr,
		pgx.NamedArgs{
			"tenantId":         config.TenantID,
			"productServiceId": config.ProductServiceID,
			"displayName":      config.DisplayName,
			"primaryColorHex":  config.PrimaryColorHex,
			"logo":             config.Logo,
			"logoContentType":  config.LogoContentType,
			"logoSha256":       config.LogoSHA256,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not insert TenantConfig: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.TenantConfig])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrAlreadyExists
	} else if err != nil {
		return nil, fmt.Errorf("could not insert TenantConfig: %w", err)
	}
	return &result, nil
}

// GetTenantLogo returns apierrors.ErrNotFound when the tenant does not exist or has no logo.
func GetTenantLogo(ctx context.Context, tenantID string) (data []byte, contentType string, sha256 string, err error) {
	db := internalctx.GetDb(ctx)
	err = db.QueryRow(ctx,
		`SELECT tc.logo, tc.logo_content_type, tc.logo_sha256
		FROM TenantConfig tc
		WHERE tc.tenant_id = @tenantId AND tc.logo IS NOT NULL`,
		pgx.NamedArgs{"tenantId": tenantID},
	).Scan(&data, &contentType, &sha256)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", "", apierrors.ErrNotFound
	} else if err != nil {
		return nil, "", "", fmt.Errorf("could not query TenantConfig logo: %w", err)
	}
	return data, contentType, sha256, nil
}
