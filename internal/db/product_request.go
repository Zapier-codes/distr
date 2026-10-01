package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/jackc/pgx/v5"
)

const productRequestOutputExpr = `
	pr.id, pr.created_at, pr.contact_email, pr.app_name, pr.theme_color
`

func CreateProductRequest(ctx context.Context, request types.ProductRequest) (*types.ProductRequest, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO ProductRequest AS pr (contact_email, app_name, theme_color)
		VALUES (@contactEmail, @appName, @themeColor)
		RETURNING`+productRequestOutputExpr,
		pgx.NamedArgs{
			"contactEmail": request.ContactEmail,
			"appName":      request.AppName,
			"themeColor":   request.ThemeColor,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not insert ProductRequest: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ProductRequest])
	if err != nil {
		return nil, fmt.Errorf("could not insert ProductRequest: %w", err)
	}
	return &result, nil
}
