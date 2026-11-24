package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/database"
	"github.com/pyrofloor/cloud-backend/internal/module/toanha"
)

type toaNhaRepository struct {
	db *pgxpool.Pool
}

func NewToaNhaNguoiDungRepository(
	db *pgxpool.Pool,
) *toaNhaRepository {
	return &toaNhaRepository{
		db: db,
	}
}

func (r *toaNhaRepository) FindByID(
	ctx context.Context,
	id string,
) (*toanha.ToaNha, error) {
	const query = `
		SELECT
			id_toa_nha,
			id_nguoi_chu,
			id_loai_mo_hinh,
			so_tang,
			ten_toa_nha,
			ngay_tao,
			dia_chi,
			ngay_cap_nhat,
			mo_ta,
			trang_thai
		FROM toa_nha
		WHERE id_toa_nha = $1
	`

	var (
		t      toanha.ToaNha
		diaChi pgtype.Text
		moTa   pgtype.Text
	)

	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.IDToaNha,
		&t.IDNguoiChu,
		&t.IDLoaiMoHinh,
		&t.SoTang,
		&t.TenToaNha,
		&t.NgayTao,
		&diaChi,
		&t.NgayCapNhat,
		&moTa,
		&t.TrangThai,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &toanha.ToaNha{}, toanha.ErrToaNhaNotFound
		}

		return &toanha.ToaNha{}, fmt.Errorf("find toa nha by id: %w", err)
	}

	if diaChi.Valid {
		t.DiaChi = diaChi.String
	}

	if moTa.Valid {
		t.MoTa = moTa.String
	}

	return &t, nil
}

func (r *toaNhaRepository) FindOwnerID(
	ctx context.Context,
	tx database.DBTX,
	buildingID string,
) (string, error) {
	const query = `
		SELECT id_nguoi_chu::text
		FROM toa_nha
		WHERE id_toa_nha = $1::uuid
		FOR UPDATE
	`
	var ownerID string
	if err := tx.QueryRow(ctx, query, buildingID).Scan(&ownerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", toanha.ErrToaNhaNotFound
		}
		return "", fmt.Errorf("find and lock building owner: %w", err)
	}
	return ownerID, nil
}

func (r *toaNhaRepository) List(
	ctx context.Context,
	filter toanha.ListFilter,
) ([]toanha.ToaNha, int64, error) {
	whereClause, args := buildListWhere(filter)

	countQuery := "SELECT COUNT(*) FROM toa_nha" + whereClause
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count toa nha: %w", err)
	}

	queryArgs := append(append([]any(nil), args...), filter.Limit, filter.Offset)
	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + 2
	query := fmt.Sprintf(`
		SELECT
			id_toa_nha,
			id_nguoi_chu,
			id_loai_mo_hinh,
			so_tang,
			ten_toa_nha,
			ngay_tao,
			dia_chi,
			ngay_cap_nhat,
			mo_ta,
			trang_thai
		FROM toa_nha%s
		ORDER BY ngay_tao DESC NULLS LAST, id_toa_nha ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list toa nha: %w", err)
	}
	defer rows.Close()

	items := make([]toanha.ToaNha, 0, filter.Limit)
	for rows.Next() {
		var (
			item      toanha.ToaNha
			createdAt pgtype.Timestamp
			updatedAt pgtype.Timestamp
			diaChi    pgtype.Text
			moTa      pgtype.Text
			trangThai pgtype.Text
		)
		if err := rows.Scan(
			&item.IDToaNha,
			&item.IDNguoiChu,
			&item.IDLoaiMoHinh,
			&item.SoTang,
			&item.TenToaNha,
			&createdAt,
			&diaChi,
			&updatedAt,
			&moTa,
			&trangThai,
		); err != nil {
			return nil, 0, fmt.Errorf("scan toa nha list row: %w", err)
		}

		if createdAt.Valid {
			item.NgayTao = createdAt.Time
		}
		if updatedAt.Valid {
			item.NgayCapNhat = updatedAt.Time
		}
		if diaChi.Valid {
			item.DiaChi = diaChi.String
		}
		if moTa.Valid {
			item.MoTa = moTa.String
		}
		if trangThai.Valid {
			item.TrangThai = trangThai.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate toa nha list rows: %w", err)
	}

	return items, total, nil
}

func buildListWhere(filter toanha.ListFilter) (string, []any) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 3)

	if filter.TrangThai != "" {
		args = append(args, filter.TrangThai)
		conditions = append(conditions, fmt.Sprintf("trang_thai = $%d", len(args)))
	}
	if filter.IDLoaiMoHinh != "" {
		args = append(args, filter.IDLoaiMoHinh)
		conditions = append(conditions, fmt.Sprintf("id_loai_mo_hinh = $%d", len(args)))
	}
	if filter.TenToaNha != "" {
		args = append(args, filter.TenToaNha)
		conditions = append(conditions, fmt.Sprintf(
			"strpos(lower(ten_toa_nha), lower($%d)) > 0",
			len(args),
		))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}
