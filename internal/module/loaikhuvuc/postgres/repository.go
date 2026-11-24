package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/module/loaikhuvuc"
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

func (r *repository) ListByModelTypeID(
	ctx context.Context,
	modelTypeID string,
) ([]loaikhuvuc.LoaiKhuVuc, error) {
	const query = `
		SELECT
			model.id_loai_mo_hinh::text,
			area_type.id_loai_khu_vuc::text,
			area_type.ten_loai_khu_vuc,
			area_type.thoi_gian_cho_xac_nhan,
			area_type.muc_do_fallback_mac_dinh,
			area_type.su_dung_gia_tri_cam_bien,
			area_type.mo_ta
		FROM loai_mo_hinh AS model
		LEFT JOIN loai_khu_vuc AS area_type
			ON area_type.id_loai_mo_hinh = model.id_loai_mo_hinh
		WHERE model.id_loai_mo_hinh = $1
		ORDER BY area_type.ten_loai_khu_vuc ASC, area_type.id_loai_khu_vuc ASC
	`
	rows, err := r.db.Query(ctx, query, modelTypeID)
	if err != nil {
		return nil, fmt.Errorf("list area types by model type ID: %w", err)
	}
	defer rows.Close()

	items := make([]loaikhuvuc.LoaiKhuVuc, 0)
	modelTypeExists := false
	for rows.Next() {
		modelTypeExists = true
		var (
			item           loaikhuvuc.LoaiKhuVuc
			areaTypeID     pgtype.Text
			confirmation   pgtype.Int4
			fallbackLevel  pgtype.Text
			useSensorValue pgtype.Bool
			description    pgtype.Text
		)
		if err := rows.Scan(
			&item.IDLoaiMoHinh,
			&areaTypeID,
			&item.TenLoaiKhuVuc,
			&confirmation,
			&fallbackLevel,
			&useSensorValue,
			&description,
		); err != nil {
			return nil, fmt.Errorf("scan area type row: %w", err)
		}

		if !areaTypeID.Valid {
			continue
		}
		item.IDLoaiKhuVuc = areaTypeID.String
		if confirmation.Valid {
			item.ThoiGianChoXacNhan = int(confirmation.Int32)
		}
		if fallbackLevel.Valid {
			item.MucDoFallbackMacDinh = fallbackLevel.String
		}
		if useSensorValue.Valid {
			item.SuDungGiaTriCamBien = useSensorValue.Bool
		}
		if description.Valid {
			item.MoTa = description.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate area type rows: %w", err)
	}
	if !modelTypeExists {
		return nil, loaikhuvuc.ErrModelTypeNotFound
	}
	return items, nil
}
