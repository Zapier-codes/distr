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

const (
	productServiceOutputExpr = `
		ps.id, ps.created_at, ps.slug, ps.type, ps.name, ps.summary, ps.description, ps.active, ps.sort_order
	`
	productServiceMediaOutputExpr = `
		psm.id, psm.product_service_id, psm.kind, psm.url, psm.caption, psm.sort_order
	`
)

// GetActiveProductServices returns the storefront entries a visitor can request, with their demo media.
func GetActiveProductServices(ctx context.Context) ([]types.ProductService, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+productServiceOutputExpr+`
		FROM ProductService ps
		WHERE ps.active
		ORDER BY ps.sort_order, ps.name`)
	if err != nil {
		return nil, fmt.Errorf("could not query ProductService: %w", err)
	}
	services, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ProductService])
	if err != nil {
		return nil, fmt.Errorf("could not query ProductService: %w", err)
	}
	if err := attachProductServiceMedia(ctx, services); err != nil {
		return nil, err
	}
	return services, nil
}

// GetActiveProductServiceBySlug returns apierrors.ErrNotFound for an unknown or inactive slug.
func GetActiveProductServiceBySlug(ctx context.Context, slug string) (*types.ProductService, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+productServiceOutputExpr+`
		FROM ProductService ps
		WHERE ps.active AND ps.slug = @slug`,
		pgx.NamedArgs{"slug": slug})
	if err != nil {
		return nil, fmt.Errorf("could not query ProductService: %w", err)
	}
	return collectOneProductService(ctx, rows)
}

// GetActiveProductServiceByID returns apierrors.ErrNotFound for an unknown or inactive id.
func GetActiveProductServiceByID(ctx context.Context, id uuid.UUID) (*types.ProductService, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+productServiceOutputExpr+`
		FROM ProductService ps
		WHERE ps.active AND ps.id = @id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return nil, fmt.Errorf("could not query ProductService: %w", err)
	}
	return collectOneProductService(ctx, rows)
}

func collectOneProductService(ctx context.Context, rows pgx.Rows) (*types.ProductService, error) {
	service, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ProductService])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not query ProductService: %w", err)
	}
	services := []types.ProductService{service}
	if err := attachProductServiceMedia(ctx, services); err != nil {
		return nil, err
	}
	return &services[0], nil
}

func attachProductServiceMedia(ctx context.Context, services []types.ProductService) error {
	if len(services) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(services))
	index := make(map[uuid.UUID]int, len(services))
	for i, service := range services {
		ids[i] = service.ID
		index[service.ID] = i
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT`+productServiceMediaOutputExpr+`
		FROM ProductServiceMedia psm
		WHERE psm.product_service_id = ANY(@ids)
		ORDER BY psm.sort_order, psm.id`,
		pgx.NamedArgs{"ids": ids})
	if err != nil {
		return fmt.Errorf("could not query ProductServiceMedia: %w", err)
	}
	media, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ProductServiceMedia])
	if err != nil {
		return fmt.Errorf("could not query ProductServiceMedia: %w", err)
	}
	for _, m := range media {
		i := index[m.ProductServiceID]
		services[i].Media = append(services[i].Media, m)
	}
	return nil
}
