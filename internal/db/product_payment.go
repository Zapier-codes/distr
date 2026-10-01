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

const productPaymentOutputExpr = `
	pp.id, pp.created_at, pp.tenant_config_id, pp.amount_minor, pp.currency, pp.bpay_payment_id, pp.payment_link,
	pp.paid_at
`

// CreateProductPayment stores the payment created for a tenant record. A record has at most one, so when another
// call stored one first this returns that one instead (and false), which keeps two simultaneous calls from handing
// out two different payment pages.
func CreateProductPayment(ctx context.Context, payment types.ProductPayment) (*types.ProductPayment, bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO ProductPayment AS pp (tenant_config_id, amount_minor, currency, bpay_payment_id, payment_link)
		VALUES (@tenantConfigId, @amountMinor, @currency, @bpayPaymentId, @paymentLink)
		ON CONFLICT (tenant_config_id) DO NOTHING
		RETURNING`+productPaymentOutputExpr,
		pgx.NamedArgs{
			"tenantConfigId": payment.TenantConfigID,
			"amountMinor":    payment.AmountMinor,
			"currency":       payment.Currency,
			"bpayPaymentId":  payment.BPayPaymentID,
			"paymentLink":    payment.PaymentLink,
		})
	if err != nil {
		return nil, false, fmt.Errorf("could not insert ProductPayment: %w", err)
	}
	created, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ProductPayment])
	if errors.Is(err, pgx.ErrNoRows) {
		existing, err := GetProductPaymentByTenantConfigID(ctx, payment.TenantConfigID)
		return existing, false, err
	} else if err != nil {
		return nil, false, fmt.Errorf("could not insert ProductPayment: %w", err)
	}
	return &created, true, nil
}

// GetProductPaymentByTenantConfigID returns apierrors.ErrNotFound when the record has no payment.
func GetProductPaymentByTenantConfigID(ctx context.Context, tenantConfigID uuid.UUID) (*types.ProductPayment, error) {
	return getProductPayment(ctx, "pp.tenant_config_id = @value", tenantConfigID)
}

// GetProductPaymentByBPayID returns apierrors.ErrNotFound for a payment id Distr did not create.
func GetProductPaymentByBPayID(ctx context.Context, bpayPaymentID string) (*types.ProductPayment, error) {
	return getProductPayment(ctx, "pp.bpay_payment_id = @value", bpayPaymentID)
}

func getProductPayment(ctx context.Context, where string, value any) (*types.ProductPayment, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+productPaymentOutputExpr+`FROM ProductPayment pp WHERE `+where,
		pgx.NamedArgs{"value": value})
	if err != nil {
		return nil, fmt.Errorf("could not query ProductPayment: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ProductPayment])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not query ProductPayment: %w", err)
	}
	return &result, nil
}

// MarkProductPaymentPaid records that B-Pay confirmed the payment, in a single statement so that of several webhooks
// racing for the same payment exactly one wins. It returns apierrors.ErrConflict when the payment was already paid.
func MarkProductPaymentPaid(ctx context.Context, id uuid.UUID) (*types.ProductPayment, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE ProductPayment AS pp SET paid_at = now()
		WHERE pp.id = @id AND pp.paid_at IS NULL
		RETURNING`+productPaymentOutputExpr,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, fmt.Errorf("could not update ProductPayment: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ProductPayment])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrConflict
	} else if err != nil {
		return nil, fmt.Errorf("could not update ProductPayment: %w", err)
	}
	return &result, nil
}

// GetProductRequest returns apierrors.ErrNotFound for an unknown id.
func GetProductRequest(ctx context.Context, id uuid.UUID) (*types.ProductRequest, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+productRequestOutputExpr+`FROM ProductRequest pr WHERE pr.id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, fmt.Errorf("could not query ProductRequest: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ProductRequest])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not query ProductRequest: %w", err)
	}
	return &result, nil
}
