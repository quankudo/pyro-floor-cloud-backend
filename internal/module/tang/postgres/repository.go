package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/module/tang"
)

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) (*repository, error) {
	if db == nil {
		return nil, errors.New("database pool is required")
	}
	return &repository{db: db}, nil
}

func (r *repository) ListByBuildingID(
	ctx context.Context,
	buildingID string,
) ([]tang.Tang, error) {
	const listQuery = `
		SELECT
			t.id_tang,
			t.so_tang
		FROM tang t
		WHERE t.id_toa_nha = $1
		ORDER BY t.so_tang DESC
	`
	rows, err := r.db.Query(ctx, listQuery, buildingID)
	if err != nil {
		return nil, fmt.Errorf("list floors by building ID: %w", err)
	}
	defer rows.Close()

	items := make([]tang.Tang, 0)
	for rows.Next() {
		var item tang.Tang
		if err := rows.Scan(&item.IDTang, &item.SoTang); err != nil {
			return nil, fmt.Errorf("scan floor row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate floor rows: %w", err)
	}
	return items, nil
}

func (r *repository) ListForBuilding(
	ctx context.Context,
	buildingID string,
) ([]tang.Tang, error) {
	return r.ListByBuildingID(ctx, buildingID)
}
