package buildtrigger

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

var (
	// ErrNotApp means the record belongs to a website product. Websites are provisioned without a build, so nothing
	// is dispatched and the record is left as it is.
	ErrNotApp = errors.New("only app products are built")
	// ErrNotQueued means the record is not waiting for a dispatch: the gate has not cleared it, or another call
	// already dispatched it.
	ErrNotQueued = errors.New("the build is not queued")
)

// DispatchTenantBuild dispatches the build of a record that the free/paid gate has cleared, meaning one in the
// queued state, and moves it to building. The gate is what makes a record queued; this function never does, so a
// request cannot reach a build without passing it.
//
// The record is claimed before GitHub is called, in a single statement, so two calls for the same record cannot both
// dispatch. When GitHub refuses, the record becomes failed with the reason. When no token is configured the record
// stays queued, so a later call can dispatch it once the instance is set up.
func DispatchTenantBuild(ctx context.Context, tenantConfigID uuid.UUID) error {
	config := env.StoreappBuild()
	if config == nil {
		return ErrNotConfigured
	}
	return dispatchTenantBuild(ctx, NewClient(*config, ""), tenantConfigID)
}

func dispatchTenantBuild(ctx context.Context, client *Client, tenantConfigID uuid.UUID) error {
	tenant, productType, err := db.GetTenantConfigForBuild(ctx, tenantConfigID)
	if err != nil {
		return err
	}
	if productType != types.ProductServiceTypeApp {
		return ErrNotApp
	}
	if tenant.BuildStatus != types.TenantBuildStatusQueued {
		return ErrNotQueued
	}

	if _, err := db.TransitionTenantBuildStatus(
		ctx, tenantConfigID, types.TenantBuildStatusQueued, types.TenantBuildStatusBuilding, nil,
	); errors.Is(err, apierrors.ErrConflict) {
		return ErrNotQueued
	} else if err != nil {
		return err
	}

	if err := client.Dispatch(ctx, tenantConfigID); err != nil {
		message := err.Error()
		if _, transitionErr := db.TransitionTenantBuildStatus(
			ctx, tenantConfigID, types.TenantBuildStatusBuilding, types.TenantBuildStatusFailed, &message,
		); transitionErr != nil {
			return fmt.Errorf("%w (and the record could not be marked failed: %v)", err, transitionErr)
		}
		return err
	}
	return nil
}
