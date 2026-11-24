package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/module/sos"
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

func (r *repository) ListCompleted(
	ctx context.Context,
	buildingID string,
	filter sos.ListFilter,
) ([]sos.SOSReport, int, error) {
	var sosType, fromTime, toTime, keyword any
	if filter.LoaiSOS != nil {
		sosType = *filter.LoaiSOS
	}
	if filter.FromTime != nil {
		fromTime = *filter.FromTime
	}
	if filter.ToTime != nil {
		toTime = *filter.ToTime
	}
	if filter.Keyword != nil {
		keyword = *filter.Keyword
	}

	const listQuery = `
		WITH latest_handler AS (
			SELECT DISTINCT ON (ls.id_bao_cao_sos)
				ls.id_bao_cao_sos,
				ls.id_nguoi_dung AS id_nguoi_phu_trach,
				nd_pt.ten_nguoi_dung AS ten_nguoi_phu_trach,
				nd_pt.so_dien_thoai AS sdt_nguoi_phu_trach,
				ls.ghi_chu AS ghi_chu_xu_ly,
				ls.ngay_tao AS thoi_gian_cap_nhat_cuoi
			FROM lich_su_xu_ly_sos ls
			JOIN nguoi_dung nd_pt ON nd_pt.id_nguoi_dung = ls.id_nguoi_dung
			WHERE ls.hanh_dong <> 'TAO_BAO_CAO_SOS'
			ORDER BY ls.id_bao_cao_sos, ls.ngay_tao DESC, ls.id_lich_su_xu_ly_sos DESC
		),
		first_ack AS (
			SELECT DISTINCT ON (ls.id_bao_cao_sos)
				ls.id_bao_cao_sos,
				ls.ngay_tao AS thoi_gian_tiep_nhan
			FROM lich_su_xu_ly_sos ls
			WHERE ls.trang_thai_moi = 'ACKNOWLEDGED'
			ORDER BY ls.id_bao_cao_sos, ls.ngay_tao ASC, ls.id_lich_su_xu_ly_sos ASC
		)
		SELECT s.id_bao_cao_sos, nd_gui.ten_nguoi_dung, nd_gui.so_dien_thoai,
			s.loai_sos, s.mo_ta, s.hinh_anh, s.id_tang, t.so_tang,
			s.id_khu_vuc, kv.ten_khu_vuc, s.ngay_tao,
			lh.thoi_gian_cap_nhat_cuoi, fa.thoi_gian_tiep_nhan,
			CASE WHEN lh.thoi_gian_cap_nhat_cuoi IS NOT NULL
				THEN EXTRACT(EPOCH FROM (lh.thoi_gian_cap_nhat_cuoi - s.ngay_tao))::bigint
				ELSE NULL END,
			s.trang_thai, lh.id_nguoi_phu_trach, lh.ten_nguoi_phu_trach,
			lh.sdt_nguoi_phu_trach, lh.ghi_chu_xu_ly
		FROM bao_cao_sos s
		JOIN nguoi_dung nd_gui ON nd_gui.id_nguoi_dung = s.id_nguoi_dung
		LEFT JOIN tang t ON t.id_tang = s.id_tang
		LEFT JOIN khu_vuc kv ON kv.id_khu_vuc = s.id_khu_vuc
		LEFT JOIN latest_handler lh ON lh.id_bao_cao_sos = s.id_bao_cao_sos
		LEFT JOIN first_ack fa ON fa.id_bao_cao_sos = s.id_bao_cao_sos
		WHERE s.id_toa_nha = $1::uuid
			AND s.trang_thai IN ('RESOLVED', 'FALSE_ALARM')
			AND ($2::varchar IS NULL OR s.loai_sos = $2)
			AND ($3::timestamp IS NULL OR s.ngay_tao >= $3)
			AND ($4::timestamp IS NULL OR s.ngay_tao < $4)
			AND ($5::text IS NULL OR nd_gui.ten_nguoi_dung ILIKE '%' || btrim($5) || '%'
				OR COALESCE(kv.ten_khu_vuc, '') ILIKE '%' || btrim($5) || '%'
				OR COALESCE(lh.ten_nguoi_phu_trach, '') ILIKE '%' || btrim($5) || '%')
		ORDER BY s.ngay_tao DESC, s.id_bao_cao_sos DESC
		LIMIT $6 OFFSET $7`

	rows, err := r.db.Query(ctx, listQuery, buildingID, sosType, fromTime, toTime, keyword, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list completed SOS reports: %w", err)
	}
	defer rows.Close()

	items := make([]sos.SOSReport, 0, filter.Limit)
	for rows.Next() {
		var item sos.SOSReport
		var idTang, idArea, handlerID pgtype.UUID
		var floorNumber pgtype.Int4
		var reportID pgtype.UUID
		var senderName, senderPhone, sosTypeValue, description, image, areaName, handlerName, handlerPhone, note pgtype.Text
		var started, ended, acknowledged pgtype.Timestamp
		var duration pgtype.Int8
		if err := rows.Scan(&reportID, &senderName, &senderPhone, &sosTypeValue, &description, &image,
			&idTang, &floorNumber, &idArea, &areaName, &started, &ended, &acknowledged, &duration,
			&item.TrangThai, &handlerID, &handlerName, &handlerPhone, &note); err != nil {
			return nil, 0, fmt.Errorf("scan completed SOS report: %w", err)
		}
		item.IDBaoCaoSOS = reportID.String()
		item.TenNguoiGui = senderName.String
		item.SDTNguoiGui = senderPhone.String
		item.LoaiSOS, item.MoTa, item.HinhAnh, item.TenKhuVuc = sosTypeValue.String, description.String, image.String, areaName.String
		if idTang.Valid {
			value := idTang.String()
			item.IDTang = &value
		}
		if floorNumber.Valid {
			value := int(floorNumber.Int32)
			item.SoTang = &value
		}
		if idArea.Valid {
			value := idArea.String()
			item.IDKhuVuc = &value
		}
		if started.Valid {
			item.ThoiGianBatDau = started.Time
		}
		if ended.Valid {
			value := ended.Time
			item.ThoiGianKetThuc = &value
		}
		if acknowledged.Valid {
			value := acknowledged.Time
			item.ThoiGianTiepNhan = &value
		}
		if duration.Valid {
			value := duration.Int64
			item.ThoiLuongXuLyGiay = &value
		}
		if handlerID.Valid {
			value := handlerID.String()
			item.IDNguoiPhuTrach = &value
		}
		item.TenNguoiPhuTrach, item.SDTNguoiPhuTrach, item.GhiChuXuLy = handlerName.String, handlerPhone.String, note.String
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate completed SOS reports: %w", err)
	}

	const countQuery = `
		WITH latest_handler AS (
			SELECT DISTINCT ON (ls.id_bao_cao_sos) ls.id_bao_cao_sos,
				nd_pt.ten_nguoi_dung AS ten_nguoi_phu_trach
			FROM lich_su_xu_ly_sos ls
			JOIN nguoi_dung nd_pt ON nd_pt.id_nguoi_dung = ls.id_nguoi_dung
			WHERE ls.hanh_dong <> 'TAO_BAO_CAO_SOS'
			ORDER BY ls.id_bao_cao_sos, ls.ngay_tao DESC, ls.id_lich_su_xu_ly_sos DESC
		)
		SELECT COUNT(*)::int
		FROM bao_cao_sos s
		JOIN nguoi_dung nd_gui ON nd_gui.id_nguoi_dung = s.id_nguoi_dung
		LEFT JOIN khu_vuc kv ON kv.id_khu_vuc = s.id_khu_vuc
		LEFT JOIN latest_handler lh ON lh.id_bao_cao_sos = s.id_bao_cao_sos
		WHERE s.id_toa_nha = $1::uuid
			AND s.trang_thai IN ('RESOLVED', 'FALSE_ALARM')
			AND ($2::varchar IS NULL OR s.loai_sos = $2)
			AND ($3::timestamp IS NULL OR s.ngay_tao >= $3)
			AND ($4::timestamp IS NULL OR s.ngay_tao < $4)
			AND ($5::text IS NULL OR nd_gui.ten_nguoi_dung ILIKE '%' || btrim($5) || '%'
				OR COALESCE(kv.ten_khu_vuc, '') ILIKE '%' || btrim($5) || '%'
				OR COALESCE(lh.ten_nguoi_phu_trach, '') ILIKE '%' || btrim($5) || '%')`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, buildingID, sosType, fromTime, toTime, keyword).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count completed SOS reports: %w", err)
	}
	return items, total, nil
}

func (r *repository) ListActive(
	ctx context.Context,
	buildingID string,
	userID string,
	filter sos.ActiveFilter,
) ([]sos.ActiveSOSReport, int, error) {
	var sosType, status any
	if filter.LoaiSOS != nil {
		sosType = *filter.LoaiSOS
	}
	if filter.TrangThai != nil {
		status = *filter.TrangThai
	}

	const listQuery = `
		SELECT
			s.id_bao_cao_sos,
			nd.ten_nguoi_dung,
			nd.so_dien_thoai,
			s.loai_sos,
			s.trang_thai,
			s.mo_ta,
			s.hinh_anh,
			s.ngay_tao,
			s.id_tang,
			t.so_tang,
			s.id_khu_vuc,
			kv.ten_khu_vuc,
			pc_match.thuoc_phan_cong_cua_toi
		FROM bao_cao_sos s
		JOIN nguoi_dung nd ON nd.id_nguoi_dung = s.id_nguoi_dung
		LEFT JOIN tang t ON t.id_tang = s.id_tang
		LEFT JOIN khu_vuc kv ON kv.id_khu_vuc = s.id_khu_vuc
		CROSS JOIN LATERAL (
			SELECT
				CASE
					WHEN s.id_tang IS NULL AND s.id_khu_vuc IS NULL THEN TRUE
					ELSE EXISTS (
						SELECT 1
						FROM phan_cong_giam_sat pc
						WHERE pc.id_nguoi_dung = $2::uuid
						  AND pc.id_toa_nha = s.id_toa_nha
						  AND (
							  (pc.id_tang IS NULL AND pc.id_khu_vuc IS NULL)
							  OR (
								  pc.id_tang = s.id_tang
								  AND (
									  pc.id_khu_vuc IS NULL
									  OR pc.id_khu_vuc = s.id_khu_vuc
								  )
							  )
						  )
					)
				END AS thuoc_phan_cong_cua_toi
		) pc_match
		WHERE s.id_toa_nha = $1::uuid
		  AND s.trang_thai IN ('NEW', 'ACKNOWLEDGED')
		  AND ($3::varchar IS NULL OR btrim($3::varchar) = '' OR s.trang_thai = $3::varchar)
		  AND ($4::varchar IS NULL OR btrim($4::varchar) = '' OR s.loai_sos = $4::varchar)
		  AND (COALESCE($5::boolean, FALSE) = FALSE OR pc_match.thuoc_phan_cong_cua_toi = TRUE)
		ORDER BY s.ngay_tao DESC, s.id_bao_cao_sos DESC
		LIMIT $6 OFFSET $7`

	rows, err := r.db.Query(
		ctx,
		listQuery,
		buildingID,
		userID,
		status,
		sosType,
		filter.ChiThuocPhanCong,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list active SOS reports: %w", err)
	}
	defer rows.Close()

	items := make([]sos.ActiveSOSReport, 0, filter.Limit)
	for rows.Next() {
		var item sos.ActiveSOSReport
		var reportID pgtype.UUID
		var idTang, idKhuVuc pgtype.UUID
		var floorNumber pgtype.Int4
		var senderName, senderPhone, sosTypeValue, statusValue pgtype.Text
		var description, image, areaName pgtype.Text
		var sentAt pgtype.Timestamp
		var assigned bool

		if err := rows.Scan(
			&reportID,
			&senderName,
			&senderPhone,
			&sosTypeValue,
			&statusValue,
			&description,
			&image,
			&sentAt,
			&idTang,
			&floorNumber,
			&idKhuVuc,
			&areaName,
			&assigned,
		); err != nil {
			return nil, 0, fmt.Errorf("scan active SOS report: %w", err)
		}

		item.IDBaoCaoSOS = reportID.String()
		item.TenNguoiGui = senderName.String
		item.SDTNguoiGui = senderPhone.String
		item.LoaiSOS = sosTypeValue.String
		item.TrangThai = statusValue.String
		item.MoTa = description.String
		item.HinhAnh = image.String
		if sentAt.Valid {
			item.ThoiGianGui = sentAt.Time
		}
		if idTang.Valid {
			value := idTang.String()
			item.IDTang = &value
		}
		if floorNumber.Valid {
			value := int(floorNumber.Int32)
			item.SoTang = &value
		}
		if idKhuVuc.Valid {
			value := idKhuVuc.String()
			item.IDKhuVuc = &value
		}
		item.TenKhuVuc = areaName.String
		item.ThuocPhanCongCuaToi = assigned
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate active SOS reports: %w", err)
	}

	const countQuery = `
		SELECT COUNT(*)::int
		FROM bao_cao_sos s
		CROSS JOIN LATERAL (
			SELECT
				CASE
					WHEN s.id_tang IS NULL AND s.id_khu_vuc IS NULL THEN TRUE
					ELSE EXISTS (
						SELECT 1
						FROM phan_cong_giam_sat pc
						WHERE pc.id_nguoi_dung = $2::uuid
						  AND pc.id_toa_nha = s.id_toa_nha
						  AND (
							  (pc.id_tang IS NULL AND pc.id_khu_vuc IS NULL)
							  OR (
								  pc.id_tang = s.id_tang
								  AND (
									  pc.id_khu_vuc IS NULL
									  OR pc.id_khu_vuc = s.id_khu_vuc
								  )
							  )
						  )
					)
				END AS thuoc_phan_cong_cua_toi
		) pc_match
		WHERE s.id_toa_nha = $1::uuid
		  AND s.trang_thai IN ('NEW', 'ACKNOWLEDGED')
		  AND ($3::varchar IS NULL OR btrim($3::varchar) = '' OR s.trang_thai = $3::varchar)
		  AND ($4::varchar IS NULL OR btrim($4::varchar) = '' OR s.loai_sos = $4::varchar)
		  AND (COALESCE($5::boolean, FALSE) = FALSE OR pc_match.thuoc_phan_cong_cua_toi = TRUE)`

	var total int
	if err := r.db.QueryRow(
		ctx,
		countQuery,
		buildingID,
		userID,
		status,
		sosType,
		filter.ChiThuocPhanCong,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count active SOS reports: %w", err)
	}
	return items, total, nil
}
