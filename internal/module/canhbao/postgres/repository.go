package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/module/canhbao"
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

func (r *repository) List(
	ctx context.Context,
	buildingID string,
	filter canhbao.ListFilter,
) ([]canhbao.AlertListItem, int, error) {
	var level, status, areaID, fromTime, toTime any
	if filter.MucDo != nil {
		level = *filter.MucDo
	}
	if filter.TrangThai != nil {
		status = *filter.TrangThai
	}
	if filter.IDKhuVuc != nil {
		areaID = *filter.IDKhuVuc
	}
	if filter.FromPhatHien != nil {
		fromTime = *filter.FromPhatHien
	}
	if filter.ToPhatHien != nil {
		toTime = *filter.ToPhatHien
	}

	const listQuery = `
		SELECT
		    cb.id_canh_bao,
		    cb.muc_do,
		    cb.trang_thai AS trang_thai_xu_ly,
		    cb.mo_ta,
		    cb.thoi_gian_phat_hien,
		    cb.thoi_gian_xac_nhan,
		    cb.thoi_gian_tiep_nhan,
		    cb.thoi_gian_ket_thuc,
		    t.id_tang,
		    t.so_tang,
		    kv.id_khu_vuc,
		    kv.ten_khu_vuc
		FROM canh_bao cb
		JOIN khu_vuc kv ON kv.id_khu_vuc = cb.id_khu_vuc
		JOIN tang t ON t.id_tang = kv.id_tang
		WHERE t.id_toa_nha = $1
		  AND ($2::TEXT IS NULL OR cb.muc_do = $2)
		  AND ($3::TEXT IS NULL OR cb.trang_thai = $3)
		  AND ($4::UUID IS NULL OR t.id_tang = $4)
		  AND ($5::UUID IS NULL OR cb.id_khu_vuc = $5)
		  AND ($6::TIMESTAMP IS NULL OR cb.thoi_gian_phat_hien >= $6)
		  AND ($7::TIMESTAMP IS NULL OR cb.thoi_gian_phat_hien <= $7)
		ORDER BY cb.thoi_gian_phat_hien DESC NULLS LAST, cb.id_canh_bao DESC
		LIMIT $8 OFFSET $9
	`
	rows, err := r.db.Query(
		ctx, listQuery, buildingID, level, status,
		filter.IDTang, areaID, fromTime, toTime,
		filter.Limit, filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	items := make([]canhbao.AlertListItem, 0, filter.Limit)
	for rows.Next() {
		var item canhbao.AlertListItem
		var description pgtype.Text
		var detectedTime pgtype.Timestamp
		var confirmedTime, acceptedTime, endedTime pgtype.Timestamp

		if err := rows.Scan(
			&item.IDCanhBao,
			&item.MucDo,
			&item.TrangThai,
			&description,
			&detectedTime,
			&confirmedTime,
			&acceptedTime,
			&endedTime,
			&item.IDTang,
			&item.SoTang,
			&item.IDKhuVuc,
			&item.TenKhuVuc,
		); err != nil {
			return nil, 0, fmt.Errorf("scan alert row: %w", err)
		}
		if description.Valid {
			item.MoTa = description.String
		}
		if detectedTime.Valid {
			item.ThoiGianPhatHien = detectedTime.Time
		}
		if confirmedTime.Valid {
			value := confirmedTime.Time
			item.ThoiGianXacNhan = &value
		}
		if acceptedTime.Valid {
			value := acceptedTime.Time
			item.ThoiGianTiepNhan = &value
		}
		if endedTime.Valid {
			value := endedTime.Time
			item.ThoiGianKetThuc = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate alert rows: %w", err)
	}

	const countQuery = `
		SELECT COUNT(*)::INT AS total
		FROM canh_bao cb
		JOIN khu_vuc kv ON kv.id_khu_vuc = cb.id_khu_vuc
		JOIN tang t ON t.id_tang = kv.id_tang
		WHERE t.id_toa_nha = $1
		  AND ($2::TEXT IS NULL OR cb.muc_do = $2)
		  AND ($3::TEXT IS NULL OR cb.trang_thai = $3)
		  AND ($4::UUID IS NULL OR t.id_tang = $4)
		  AND ($5::UUID IS NULL OR cb.id_khu_vuc = $5)
		  AND ($6::TIMESTAMP IS NULL OR cb.thoi_gian_phat_hien >= $6)
		  AND ($7::TIMESTAMP IS NULL OR cb.thoi_gian_phat_hien <= $7)
	`
	var total int
	if err := r.db.QueryRow(
		ctx, countQuery, buildingID, level, status,
		filter.IDTang, areaID, fromTime, toTime,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}
	return items, total, nil
}

func (r *repository) ListActive(
	ctx context.Context,
	buildingID string,
	userID string,
	filter canhbao.ActiveFilter,
) ([]canhbao.ActiveAlertListItem, int, error) {
	var status any
	if filter.TrangThai != nil {
		status = *filter.TrangThai
	}

	const listQuery = `
		SELECT
		    cb.id_canh_bao,
		    cb.muc_do,
		    cb.trang_thai,
		    cb.mo_ta,
		    cb.thoi_gian_phat_hien,
		    cb.thoi_gian_xac_nhan,
		    cb.thoi_gian_tiep_nhan,
		    t.id_tang,
		    t.so_tang,
		    kv.id_khu_vuc,
		    kv.ten_khu_vuc,
		    pc_match.thuoc_phan_cong_cua_toi,
		    ack_user.id_nguoi_tiep_nhan,
		    ack_user.ten_nguoi_tiep_nhan,
		    ack_user.sdt_nguoi_tiep_nhan,
		    ack_user.thoi_gian_thao_tac_tiep_nhan,
		    COALESCE(events_agg.danh_sach_su_kien, '[]'::jsonb)
		FROM canh_bao cb
		JOIN khu_vuc kv ON kv.id_khu_vuc = cb.id_khu_vuc
		JOIN tang t ON t.id_tang = kv.id_tang
		CROSS JOIN LATERAL (
		    SELECT EXISTS (
		        SELECT 1
		        FROM phan_cong_giam_sat pc
		        WHERE pc.id_nguoi_dung = $2::uuid
		          AND pc.id_toa_nha = t.id_toa_nha
		          AND (
		              (pc.id_tang IS NULL AND pc.id_khu_vuc IS NULL)
		              OR (pc.id_tang = t.id_tang AND pc.id_khu_vuc IS NULL)
		              OR (pc.id_khu_vuc = cb.id_khu_vuc)
		          )
		    ) AS thuoc_phan_cong_cua_toi
		) pc_match
		LEFT JOIN LATERAL (
		    SELECT DISTINCT ON (nk.id_canh_bao)
		        nk.id_nguoi_thuc_hien AS id_nguoi_tiep_nhan,
		        nd.ten_nguoi_dung AS ten_nguoi_tiep_nhan,
		        nd.so_dien_thoai AS sdt_nguoi_tiep_nhan,
		        nk.ngay_tao AS thoi_gian_thao_tac_tiep_nhan
		    FROM nhat_ky_xu_ly_canh_bao nk
		    LEFT JOIN nguoi_dung nd ON nd.id_nguoi_dung = nk.id_nguoi_thuc_hien
		    WHERE nk.id_canh_bao = cb.id_canh_bao
		      AND nk.loai_hanh_dong IN ('TIEP_NHAN_CANH_BAO', 'ACKNOWLEDGE')
		    ORDER BY nk.id_canh_bao, nk.ngay_tao DESC
		) ack_user ON TRUE
		LEFT JOIN LATERAL (
		    SELECT jsonb_agg(
		        jsonb_build_object(
		            'id_su_kien_canh_bao', sk.id_su_kien_canh_bao,
		            'id_cam_bien', sk.id_cam_bien,
		            'ma_cam_bien', sk.ma_cam_bien,
		            'loai_cam_bien', sk.loai_cam_bien,
		            'don_vi_do', sk.don_vi_do,
		            'vi_tri_lap_dat', sk.vi_tri_lap_dat,
		            'loai_su_kien', sk.loai_su_kien,
		            'gia_tri_bat_dau', sk.gia_tri,
		            'nguong', sk.nguong,
		            'muc_do', sk.muc_do,
		            'thoi_gian_phat_hien', sk.thoi_gian_phat_hien
		        )
		        ORDER BY sk.thoi_gian_phat_hien ASC, sk.id_su_kien_canh_bao ASC
		    ) AS danh_sach_su_kien
		    FROM (
		        SELECT
		            sk_inner.id_su_kien_canh_bao,
		            sk_inner.id_cam_bien,
		            sk_inner.loai_su_kien,
		            sk_inner.gia_tri,
		            sk_inner.nguong,
		            sk_inner.muc_do,
		            sk_inner.thoi_gian_phat_hien,
		            cb_s.ma_cam_bien,
		            cb_s.loai_cam_bien,
		            cb_s.don_vi_do,
		            cb_s.vi_tri_lap_dat
		        FROM su_kien_canh_bao sk_inner
		        JOIN cam_bien cb_s ON cb_s.id_cam_bien = sk_inner.id_cam_bien
		        WHERE sk_inner.id_canh_bao = cb.id_canh_bao
		        ORDER BY sk_inner.thoi_gian_phat_hien ASC, sk_inner.id_su_kien_canh_bao ASC
		        LIMIT 3
		    ) sk
		) events_agg ON TRUE
		WHERE t.id_toa_nha = $1::uuid
		  AND cb.trang_thai IN ('NEW', 'ACKNOWLEDGED', 'CONFIRMED')
		  AND ($3::varchar IS NULL OR btrim($3::varchar) = '' OR cb.trang_thai = $3::varchar)
		  AND (COALESCE($4::boolean, FALSE) = FALSE OR pc_match.thuoc_phan_cong_cua_toi = TRUE)
		ORDER BY cb.thoi_gian_phat_hien DESC
		LIMIT $5 OFFSET $6
	`

	rows, err := r.db.Query(ctx, listQuery, buildingID, userID, status, filter.ChiThuocPhanCong, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list active alerts: %w", err)
	}
	defer rows.Close()

	items := make([]canhbao.ActiveAlertListItem, 0, filter.Limit)
	for rows.Next() {
		var item canhbao.ActiveAlertListItem
		var alertID, floorID, areaID, ackUserID pgtype.UUID
		var floorNumber pgtype.Int4
		var description, level, statusValue, areaName, ackName, ackPhone pgtype.Text
		var detected, confirmed, accepted, ackOperation pgtype.Timestamp
		var assigned bool
		var eventsJSON []byte

		if err := rows.Scan(
			&alertID, &level, &statusValue, &description,
			&detected, &confirmed, &accepted,
			&floorID, &floorNumber, &areaID, &areaName,
			&assigned, &ackUserID, &ackName, &ackPhone, &ackOperation,
			&eventsJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("scan active alert row: %w", err)
		}

		item.IDCanhBao = alertID.String()
		item.MucDo = level.String
		item.TrangThai = statusValue.String
		item.MoTa = description.String
		item.SoTang = int(floorNumber.Int32)
		item.IDTang = floorID.String()
		item.IDKhuVuc = areaID.String()
		item.TenKhuVuc = areaName.String
		item.ThuocPhanCongCuaToi = assigned
		if detected.Valid {
			item.ThoiGianPhatHien = detected.Time
		}
		if confirmed.Valid {
			value := confirmed.Time
			item.ThoiGianXacNhan = &value
		}
		if accepted.Valid {
			value := accepted.Time
			item.ThoiGianTiepNhan = &value
		}
		if ackUserID.Valid {
			value := ackUserID.String()
			item.IDNguoiTiepNhan = &value
		}
		item.TenNguoiTiepNhan = ackName.String
		item.SDTNguoiTiepNhan = ackPhone.String
		if ackOperation.Valid {
			value := ackOperation.Time
			item.ThoiGianThaoTacTiepNhan = &value
		}

		if len(eventsJSON) == 0 {
			item.DanhSachSuKien = json.RawMessage(`[]`)
		} else {
			item.DanhSachSuKien = json.RawMessage(eventsJSON)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate active alert rows: %w", err)
	}

	const countQuery = `
		SELECT COUNT(*)::int
		FROM canh_bao cb
		JOIN khu_vuc kv ON kv.id_khu_vuc = cb.id_khu_vuc
		JOIN tang t ON t.id_tang = kv.id_tang
		WHERE t.id_toa_nha = $1::uuid
		  AND cb.trang_thai IN ('NEW', 'ACKNOWLEDGED', 'CONFIRMED')
		  AND ($3::varchar IS NULL OR btrim($3::varchar) = '' OR cb.trang_thai = $3::varchar)
		  AND (
		      COALESCE($4::boolean, FALSE) = FALSE
		      OR EXISTS (
		          SELECT 1
		          FROM phan_cong_giam_sat pc
		          WHERE pc.id_nguoi_dung = $2::uuid
		            AND pc.id_toa_nha = t.id_toa_nha
		            AND (
		                (pc.id_tang IS NULL AND pc.id_khu_vuc IS NULL)
		                OR (pc.id_tang = t.id_tang AND pc.id_khu_vuc IS NULL)
		                OR (pc.id_khu_vuc = cb.id_khu_vuc)
		            )
		      )
		  )
	`
	var total int
	if err := r.db.QueryRow(
		ctx,
		countQuery,
		buildingID,
		userID,
		status,
		filter.ChiThuocPhanCong,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count active alerts: %w", err)
	}
	return items, total, nil
}
