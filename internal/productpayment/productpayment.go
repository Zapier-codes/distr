// Package productpayment is the paid half of the free/paid gate (f.xiii): it creates the one-time B-Pay payment of a
// request the free tier did not cover, and, once B-Pay confirms the payment, queues the build.
package productpayment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/bpay"
	"github.com/distr-sh/distr/internal/buildtrigger"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"go.uber.org/zap"
)

var (
	// ErrNoPrice means the product has no price, so it cannot be paid for.
	ErrNoPrice = errors.New("the product has no price")
	// ErrNotAwaitingPayment means the record is not waiting for payment: it was cleared already, or paid.
	ErrNotAwaitingPayment = errors.New("the request is not awaiting payment")
	// ErrUnknownPayment means the payment was not created by this instance. The B-Pay account may serve other
	// systems too, so this is acknowledged, not an error of the sender.
	ErrUnknownPayment = errors.New("the payment is not one of this instance")
	// ErrNotSucceeded means B-Pay does not report the payment as succeeded, whatever the webhook claimed.
	ErrNotSucceeded = errors.New("the payment has not succeeded")
	// ErrMismatch means the payment B-Pay reports does not cover what was asked for.
	ErrMismatch = errors.New("the payment does not match the amount that was asked for")
)

// EnsurePaymentLink returns the B-Pay page that takes the payment of a record awaiting it, creating the payment on the
// first call. Repeating the call returns the same page. Nothing about the requester is sent to B-Pay.
func EnsurePaymentLink(ctx context.Context, tenant *types.TenantConfig, service *types.ProductService) (string, error) {
	client, err := bpay.FromEnv()
	if err != nil {
		return "", err
	}
	if service.PriceMinor == nil || service.PriceCurrency == nil {
		return "", ErrNoPrice
	}
	if tenant.BuildStatus != types.TenantBuildStatusAwaitingGate {
		return "", ErrNotAwaitingPayment
	}

	existing, err := db.GetProductPaymentByTenantConfigID(ctx, tenant.ID)
	if err == nil {
		return existingLink(existing)
	} else if !errors.Is(err, apierrors.ErrNotFound) {
		return "", err
	}

	returnURL := publicBaseURL(env.Host(), env.HostScheme()) + "/store"
	payment, err := client.CreatePayment(
		ctx, int64(*service.PriceMinor), *service.PriceCurrency, returnURL, service.Name,
		map[string]string{"distr_tenant_config_id": tenant.ID.String()},
	)
	if err != nil {
		return "", err
	}

	stored, _, err := db.CreateProductPayment(ctx, types.ProductPayment{
		TenantConfigID: tenant.ID,
		AmountMinor:    *service.PriceMinor,
		Currency:       *service.PriceCurrency,
		BPayPaymentID:  payment.ID,
		PaymentLink:    payment.Link,
	})
	if err != nil {
		return "", err
	}
	return existingLink(stored)
}

func existingLink(payment *types.ProductPayment) (string, error) {
	if payment.PaidAt != nil {
		return "", ErrNotAwaitingPayment
	}
	return payment.PaymentLink, nil
}

// ConfirmPaid handles B-Pay's word that a payment succeeded. The word is only a hint: the payment is retrieved from
// B-Pay and has to be succeeded and cover exactly the amount and currency stored when it was created. Then, in one
// transaction, the payment is marked paid and the record moves from awaiting_gate to queued, and, for an app, the
// build is dispatched. It is safe to call again for the same payment: a payment already paid does nothing.
func ConfirmPaid(ctx context.Context, bpayPaymentID string) error {
	log := internalctx.GetLogger(ctx)
	client, err := bpay.FromEnv()
	if err != nil {
		return err
	}

	payment, err := db.GetProductPaymentByBPayID(ctx, bpayPaymentID)
	if errors.Is(err, apierrors.ErrNotFound) {
		return ErrUnknownPayment
	} else if err != nil {
		return err
	}
	if payment.PaidAt != nil {
		return nil
	}

	reported, err := client.GetPayment(ctx, payment.BPayPaymentID)
	if err != nil {
		return err
	}
	if reported.ID != payment.BPayPaymentID {
		return fmt.Errorf("%w: B-Pay answered another payment", ErrMismatch)
	}
	if !strings.EqualFold(reported.Status, bpay.StatusSucceeded) {
		return ErrNotSucceeded
	}
	if reported.AmountMinor != int64(payment.AmountMinor) || !strings.EqualFold(reported.Currency, payment.Currency) {
		return ErrMismatch
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		if _, err := db.MarkProductPaymentPaid(ctx, payment.ID); err != nil {
			return err
		}
		// The record may already have moved on, which is not a reason to undo the payment: the money is received.
		if _, err := db.TransitionTenantBuildStatus(
			ctx, payment.TenantConfigID, types.TenantBuildStatusAwaitingGate, types.TenantBuildStatusQueued, nil,
		); err != nil && !errors.Is(err, apierrors.ErrConflict) {
			return err
		}
		return nil
	})
	if errors.Is(err, apierrors.ErrConflict) {
		// A concurrent confirmation of the same payment won, and does the rest.
		return nil
	} else if err != nil {
		return err
	}

	// The money is received and the record is queued, so a dispatch that cannot be made leaves it queued for the
	// operator and is not an error to answer to B-Pay, which would only resend the webhook.
	ctx = context.WithoutCancel(ctx)
	if err := buildtrigger.DispatchTenantBuild(ctx, payment.TenantConfigID); errors.Is(err, buildtrigger.ErrNotApp) {
		// A website is provisioned without a build.
	} else if err != nil {
		log.Warn("could not dispatch the build of a paid request",
			zap.String("tenantConfigId", payment.TenantConfigID.String()), zap.Error(err))
	}
	return nil
}

func publicBaseURL(host string, scheme env.URLScheme) string {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if strings.Contains(host, "://") {
		return host
	}
	return fmt.Sprintf("%v://%v", scheme, host)
}
