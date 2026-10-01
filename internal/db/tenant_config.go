package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const tenantConfigOutputExpr = `
	tc.id, tc.created_at, tc.tenant_id, tc.product_service_id, tc.display_name, tc.primary_color_hex,
	tc.logo, tc.logo_content_type, tc.logo_sha256, tc.cdn_base, tc.catalog_index_base_url, tc.domains,
	tc.sequence, tc.build_status, tc.build_status_updated_at, tc.build_status_message,
	tc.release_repository, tc.release_id, tc.release_asset_id
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

// CompleteTenantBuild ends a running build, in a single statement, so that of several reports racing for the same
// record exactly one wins. to is succeeded, with the release asset, or failed, with the reason. It returns
// apierrors.ErrConflict when the record is not building, or does not exist.
func CompleteTenantBuild(
	ctx context.Context, id uuid.UUID, to types.TenantBuildStatus, message *string,
	repository string, releaseID, releaseAssetID int64,
) (*types.TenantConfig, error) {
	var repositoryArg *string
	var releaseIDArg, releaseAssetIDArg *int64
	if to == types.TenantBuildStatusSucceeded {
		repositoryArg, releaseIDArg, releaseAssetIDArg = &repository, &releaseID, &releaseAssetID
	}

	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE TenantConfig AS tc SET
			build_status = @to,
			build_status_message = @message,
			build_status_updated_at = now(),
			release_repository = @repository,
			release_id = @releaseId,
			release_asset_id = @releaseAssetId
		WHERE tc.id = @id AND tc.build_status = @building
		RETURNING`+tenantConfigOutputExpr,
		pgx.NamedArgs{
			"id": id, "to": to, "message": message, "building": types.TenantBuildStatusBuilding,
			"repository": repositoryArg, "releaseId": releaseIDArg, "releaseAssetId": releaseAssetIDArg,
		})
	if err != nil {
		return nil, fmt.Errorf("could not complete TenantConfig build: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.TenantConfig])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrConflict
	} else if err != nil {
		return nil, fmt.Errorf("could not complete TenantConfig build: %w", err)
	}
	return &result, nil
}

// BuildReadyRecipient is who the mail about a finished build goes to.
type BuildReadyRecipient struct {
	ContactEmail string
	AppName      string
	// EmailSentAt is nil until Novu accepted the mail.
	EmailSentAt *time.Time
}

// GetBuildReadyRecipient returns apierrors.ErrNotFound when no request is attached to the record.
func GetBuildReadyRecipient(ctx context.Context, tenantConfigID uuid.UUID) (*BuildReadyRecipient, error) {
	db := internalctx.GetDb(ctx)
	var result BuildReadyRecipient
	err := db.QueryRow(ctx,
		`SELECT pr.contact_email, pr.app_name, tc.build_email_sent_at
		FROM ProductRequest pr
		JOIN TenantConfig tc ON tc.id = pr.tenant_config_id
		WHERE tc.id = @id`,
		pgx.NamedArgs{"id": tenantConfigID},
	).Scan(&result.ContactEmail, &result.AppName, &result.EmailSentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not query build mail recipient: %w", err)
	}
	return &result, nil
}

// SetTenantBuildDownloadToken stores the SHA-256 of a new download token, replacing the previous one, for a record
// whose build succeeded and whose mail has not been accepted yet. It returns apierrors.ErrConflict otherwise.
func SetTenantBuildDownloadToken(
	ctx context.Context, id uuid.UUID, tokenHash []byte, expiresAt time.Time,
) error {
	db := internalctx.GetDb(ctx)
	tag, err := db.Exec(ctx,
		`UPDATE TenantConfig SET download_token_hash = @hash, download_token_expires_at = @expiresAt
		WHERE id = @id AND build_status = @succeeded AND build_email_sent_at IS NULL`,
		pgx.NamedArgs{
			"id": id, "hash": tokenHash, "expiresAt": expiresAt, "succeeded": types.TenantBuildStatusSucceeded,
		})
	if err != nil {
		return fmt.Errorf("could not store download token: %w", err)
	} else if tag.RowsAffected() == 0 {
		return apierrors.ErrConflict
	}
	return nil
}

// MarkTenantBuildEmailSent records that Novu accepted the mail.
func MarkTenantBuildEmailSent(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(ctx,
		`UPDATE TenantConfig SET build_email_sent_at = now() WHERE id = @id`,
		pgx.NamedArgs{"id": id},
	); err != nil {
		return fmt.Errorf("could not record that the build mail was sent: %w", err)
	}
	return nil
}

// GetTenantConfigByDownloadToken returns the record of a succeeded build whose download token has the given hash
// and has not expired. It returns apierrors.ErrNotFound otherwise, without saying which of those it was.
func GetTenantConfigByDownloadToken(ctx context.Context, tokenHash []byte) (*types.TenantConfig, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+tenantConfigOutputExpr+`
		FROM TenantConfig tc
		WHERE tc.download_token_hash = @hash
			AND tc.download_token_expires_at > now()
			AND tc.build_status = @succeeded
			AND tc.release_repository IS NOT NULL`,
		pgx.NamedArgs{"hash": tokenHash, "succeeded": types.TenantBuildStatusSucceeded})
	if err != nil {
		return nil, fmt.Errorf("could not query TenantConfig by download token: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.TenantConfig])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not query TenantConfig by download token: %w", err)
	}
	return &result, nil
}
