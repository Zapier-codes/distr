package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ClaimFreeTier records that the device behind fingerprintHash used its free product on the given tenant record. It
// reports whether this call made the claim: false means the device had already claimed one. The check and the insert
// are one statement, so of several requests racing from one device exactly one wins, and a conflict does not abort an
// enclosing transaction.
func ClaimFreeTier(ctx context.Context, fingerprintHash []byte, tenantConfigID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	tag, err := db.Exec(ctx,
		`INSERT INTO FreeTierClaim (fingerprint_hash, tenant_config_id)
		VALUES (@fingerprintHash, @tenantConfigId)
		ON CONFLICT (fingerprint_hash) DO NOTHING`,
		pgx.NamedArgs{"fingerprintHash": fingerprintHash, "tenantConfigId": tenantConfigID})
	if err != nil {
		return false, fmt.Errorf("could not insert FreeTierClaim: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
