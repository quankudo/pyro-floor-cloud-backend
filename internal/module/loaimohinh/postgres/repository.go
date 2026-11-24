package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/module/loaimohinh"
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

func (r *repository) List(ctx context.Context) ([]loaimohinh.LoaiMoHinh, error) {
	const query = `
		SELECT
			id_loai_mo_hinh,
			ten_loai_mo_hinh,
			mo_ta
		FROM loai_mo_hinh
		ORDER BY ten_loai_mo_hinh ASC, id_loai_mo_hinh ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list model types: %w", err)
	}
	defer rows.Close()

	items := make([]loaimohinh.LoaiMoHinh, 0)
	for rows.Next() {
		var (
			item loaimohinh.LoaiMoHinh
			note pgtype.Text
		)
		if err := rows.Scan(
			&item.IDLoaiMoHinh,
			&item.TenLoaiMoHinh,
			&note,
		); err != nil {
			return nil, fmt.Errorf("scan model type row: %w", err)
		}
		if note.Valid {
			item.MoTa = note.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model type rows: %w", err)
	}
	return items, nil
}
