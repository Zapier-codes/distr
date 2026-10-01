package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const tenantConfigOutputExpr = `
	tc.id, tc.created_at, tc.tenant_id, tc.product_service_id, tc.display_name, tc.primary_color_hex,
	tc.logo, tc.logo_content_type, tc.logo_sha256, tc.cdn_base, tc.catalog_index_base_url, tc.domains,
	tc.sequence, tc.build_status, tc.build_status_updated_at, tc.build_status_message
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

// GetTenantConfigForBuild returns the record together with the type of its product, which decides whether a build
// is dispatched at all. It returns apierrors.ErrNotFound for an unknown id.
func GetTenantConfigForBuild(
	ctx context.Context, id uuid.UUID,
) (*types.TenantConfig, types.ProductServiceType, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+tenantConfigOutputExpr+`
		FROM TenantConfig tc
		WHERE tc.id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, "", fmt.Errorf("could not query TenantConfig: %w", err)
	}
	tenant, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.TenantConfig])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", apierrors.ErrNotFound
	} else if err != nil {
		return nil, "", fmt.Errorf("could not query TenantConfig: %w", err)
	}

	var productType types.ProductServiceType
	if err := db.QueryRow(ctx,
		`SELECT ps.type FROM ProductService ps WHERE ps.id = @id`,
		pgx.NamedArgs{"id": tenant.ProductServiceID},
	).Scan(&productType); err != nil {
		return nil, "", fmt.Errorf("could not query ProductService type: %w", err)
	}
	return &tenant, productType, nil
}

// TransitionTenantBuildStatus moves a record from one build status to another in a single statement, so that of
// several callers racing for the same transition exactly one wins. It returns apierrors.ErrConflict when the record
// is not in the from status, or does not exist.
func TransitionTenantBuildStatus(
	ctx context.Context, id uuid.UUID, from, to types.TenantBuildStatus, message *string,
) (*types.TenantConfig, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE TenantConfig AS tc SET
			build_status = @to,
			build_status_message = @message,
			build_status_updated_at = now()
		WHERE tc.id = @id AND tc.build_status = @from
		RETURNING`+tenantConfigOutputExpr,
		pgx.NamedArgs{"id": id, "from": from, "to": to, "message": message})
	if err != nil {
		return nil, fmt.Errorf("could not update TenantConfig build status: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.TenantConfig])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrConflict
	} else if err != nil {
		return nil, fmt.Errorf("could not update TenantConfig build status: %w", err)
	}
	return &result, nil
}
