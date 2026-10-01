package types

import (
	"time"

	"github.com/google/uuid"
)

// ProductPayment is the one-time charge made for a tenant record through B-Pay.
type ProductPayment struct {
	ID             uuid.UUID `db:"id"`
	CreatedAt      time.Time `db:"created_at"`
	TenantConfigID uuid.UUID `db:"tenant_config_id"`
	// AmountMinor and Currency are what the payment has to cover, copied from the product when it was created.
	AmountMinor   int    `db:"amount_minor"`
	Currency      string `db:"currency"`
	BPayPaymentID string `db:"bpay_payment_id"`
	PaymentLink   string `db:"payment_link"`
	// PaidAt is nil until B-Pay has confirmed the payment as succeeded.
	PaidAt *time.Time `db:"paid_at"`
}
