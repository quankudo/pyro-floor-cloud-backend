package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/module/khuvuc"
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

func (r *repository) ListByFloorID(
	ctx context.Context,
	floorID string,
) ([]khuvuc.KhuVuc, error) {
	const floorExistsQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM tang
			WHERE id_tang = $1
		)
	`
	var floorExists bool
	if err := r.db.QueryRow(ctx, floorExistsQuery, floorID).Scan(&floorExists); err != nil {
		return nil, fmt.Errorf("check floor existence: %w", err)
	}
	if !floorExists {
		return nil, khuvuc.ErrFloorNotFound
	}

	const query = `
		SELECT
			id_khu_vuc,
			id_tang,
			id_loai_khu_vuc,
			ten_khu_vuc,
			mo_ta
		FROM khu_vuc
		WHERE id_tang = $1
		ORDER BY ten_khu_vuc ASC, id_khu_vuc ASC
	`
	rows, err := r.db.Query(ctx, query, floorID)
	if err != nil {
		return nil, fmt.Errorf("list areas by floor ID: %w", err)
	}
	defer rows.Close()

	items := make([]khuvuc.KhuVuc, 0)
	for rows.Next() {
		var (
			item khuvuc.KhuVuc
			note pgtype.Text
		)
		if err := rows.Scan(
			&item.IDKhuVuc,
			&item.IDTang,
			&item.IDLoaiKhuVuc,
			&item.TenKhuVuc,
			&note,
		); err != nil {
			return nil, fmt.Errorf("scan area row: %w", err)
		}
		if note.Valid {
			item.MoTa = note.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate area rows: %w", err)
	}
	return items, nil
}

func (r *repository) ListShortByFloorID(
	ctx context.Context,
	floorID string,
) ([]khuvuc.KhuVucShort, error) {
	const query = `
		SELECT
			kv.id_khu_vuc,
			kv.ten_khu_vuc,
			lkv.id_loai_khu_vuc,
			lkv.ten_loai_khu_vuc
		FROM khu_vuc kv
		JOIN loai_khu_vuc lkv
		  ON lkv.id_loai_khu_vuc = kv.id_loai_khu_vuc
		WHERE kv.id_tang = $1
		ORDER BY kv.ten_khu_vuc ASC
	`
	rows, err := r.db.Query(ctx, query, floorID)
	if err != nil {
		return nil, fmt.Errorf("list short area data by floor ID: %w", err)
	}
	defer rows.Close()

	items := make([]khuvuc.KhuVucShort, 0)
	for rows.Next() {
		var item khuvuc.KhuVucShort
		if err := rows.Scan(
			&item.IDKhuVuc,
			&item.TenKhuVuc,
			&item.IDLoaiKhuVuc,
			&item.TenLoaiKhuVuc,
		); err != nil {
			return nil, fmt.Errorf("scan short area row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate short area rows: %w", err)
	}
	return items, nil
}

func (r *repository) ListMonitoring(
	ctx context.Context,
	buildingID string,
	filter khuvuc.MonitoringFilter,
) ([]khuvuc.RoomMonitoring, error) {
	const query = `
		WITH target_floors AS (
			SELECT t.id_tang, t.so_tang, t.id_toa_nha
			FROM tang t
			WHERE t.id_toa_nha = $1::uuid
			  AND ($2::int IS NULL OR t.so_tang = $2::int)
			ORDER BY t.so_tang ASC
			LIMIT CASE WHEN $2::int IS NULL THEN 5 ELSE 1000 END
			OFFSET CASE WHEN $2::int IS NULL THEN COALESCE($3::int, 0) ELSE 0 END
		),
		latest_sensors AS (
			SELECT tb.id_khu_vuc,
				MAX(CASE
					WHEN tb.trang_thai != 'ONLINE' OR cam.trang_thai != 'ACTIVE' THEN NULL
					WHEN cam.loai_cam_bien = 'NHIET_DO' THEN cam.gia_tri_hien_tai
				END)::float8 AS nhiet_do,
				MAX(CASE WHEN cam.loai_cam_bien = 'NHIET_DO' THEN cam.don_vi_do END) AS don_vi_nhiet_do,
				MAX(CASE
					WHEN tb.trang_thai != 'ONLINE' OR cam.trang_thai != 'ACTIVE' THEN NULL
					WHEN cam.loai_cam_bien = 'KHOI' THEN cam.gia_tri_hien_tai
				END)::float8 AS nong_do_khoi,
				MAX(CASE WHEN cam.loai_cam_bien = 'KHOI' THEN cam.don_vi_do END) AS don_vi_khoi
			FROM thiet_bi tb
			JOIN cam_bien cam ON tb.id_thiet_bi = cam.id_thiet_bi
			JOIN khu_vuc kv ON tb.id_khu_vuc = kv.id_khu_vuc
			JOIN target_floors tf ON kv.id_tang = tf.id_tang
			GROUP BY tb.id_khu_vuc
		),
		area_device_hardware_stats AS (
			SELECT tb.id_khu_vuc,
				COUNT(*) AS total_devices,
				SUM(CASE WHEN tb.trang_thai != 'ONLINE' THEN 1 ELSE 0 END) AS offline_devices
			FROM thiet_bi tb
			JOIN khu_vuc kv ON tb.id_khu_vuc = kv.id_khu_vuc
			JOIN target_floors tf ON kv.id_tang = tf.id_tang
			GROUP BY tb.id_khu_vuc
		),
		area_sensor_hardware_stats AS (
			SELECT tb.id_khu_vuc,
				SUM(CASE WHEN cam.trang_thai != 'ACTIVE' THEN 1 ELSE 0 END) AS error_sensors
			FROM cam_bien cam
			JOIN thiet_bi tb ON cam.id_thiet_bi = tb.id_thiet_bi
			JOIN khu_vuc kv ON tb.id_khu_vuc = kv.id_khu_vuc
			JOIN target_floors tf ON kv.id_tang = tf.id_tang
			GROUP BY tb.id_khu_vuc
		),
		latest_active_alerts AS (
			SELECT cb.id_khu_vuc, cb.muc_do, cb.trang_thai, cb.mo_ta,
			       cb.thoi_gian_phat_hien,
			       ROW_NUMBER() OVER (
			         PARTITION BY cb.id_khu_vuc
			         ORDER BY cb.thoi_gian_phat_hien DESC
			       ) AS rn
			FROM canh_bao cb
			JOIN khu_vuc kv ON cb.id_khu_vuc = kv.id_khu_vuc
			JOIN target_floors tf ON kv.id_tang = tf.id_tang
			WHERE cb.trang_thai NOT IN ('RESOLVED', 'FALSE_ALARM')
		),
		area_enriched AS (
			SELECT tf.so_tang, tf.id_tang, kv.id_khu_vuc, kv.ten_khu_vuc, kv.mo_ta,
			       lk.ten_loai_khu_vuc,
			       cb.muc_do AS muc_do_canh_bao,
			       cb.trang_thai AS trang_thai_canh_bao,
			       cb.mo_ta AS mo_ta_canh_bao,
			       (cb.id_khu_vuc IS NOT NULL) AS co_canh_bao,
			       COALESCE(adh.total_devices, 0) AS total_devices,
			       COALESCE(adh.offline_devices, 0) AS offline_devices,
			       COALESCE(ash.error_sensors, 0) AS error_sensors,
			       (COALESCE(adh.offline_devices, 0) > 0 OR COALESCE(ash.error_sensors, 0) > 0) AS co_loi_phan_cung,
			       CASE
			         WHEN COALESCE(adh.total_devices, 0) = 0 THEN 'NO_DEVICE'
			         WHEN COALESCE(adh.offline_devices, 0) > 0 THEN 'DEVICE_OFFLINE'
			         WHEN COALESCE(ash.error_sensors, 0) > 0 THEN 'SENSOR_ERROR'
			         ELSE 'ONLINE'
			       END AS trang_thai_phan_cung,
			       ls.nhiet_do,
			       COALESCE(ls.don_vi_nhiet_do, '°C') AS don_vi_nhiet_do,
			       ls.nong_do_khoi,
			       COALESCE(ls.don_vi_khoi, 'ppm') AS don_vi_khoi,
			       CASE
			         WHEN COALESCE(adh.total_devices, 0) = 0 THEN 'Chưa có thiết bị'
			         WHEN cb.id_khu_vuc IS NOT NULL THEN 'Đang có cảnh báo nguy hiểm'
			         WHEN COALESCE(adh.offline_devices, 0) > 0 OR COALESCE(ash.error_sensors, 0) > 0 THEN 'Mất kết nối / Lỗi thiết bị phần cứng'
			         ELSE 'Hoạt động chuẩn'
			       END AS mo_ta_tinh_trang
			FROM target_floors tf
			JOIN khu_vuc kv ON tf.id_tang = kv.id_tang
			JOIN loai_khu_vuc lk ON kv.id_loai_khu_vuc = lk.id_loai_khu_vuc
			LEFT JOIN latest_sensors ls ON kv.id_khu_vuc = ls.id_khu_vuc
			LEFT JOIN area_device_hardware_stats adh ON kv.id_khu_vuc = adh.id_khu_vuc
			LEFT JOIN area_sensor_hardware_stats ash ON kv.id_khu_vuc = ash.id_khu_vuc
			LEFT JOIN latest_active_alerts cb ON kv.id_khu_vuc = cb.id_khu_vuc AND cb.rn = 1
		)
		SELECT so_tang, id_tang, id_khu_vuc, ten_khu_vuc, mo_ta,
		       ten_loai_khu_vuc, muc_do_canh_bao, trang_thai_canh_bao,
		       mo_ta_canh_bao, co_canh_bao, co_loi_phan_cung,
		       trang_thai_phan_cung, nhiet_do, don_vi_nhiet_do,
		       nong_do_khoi, don_vi_khoi, mo_ta_tinh_trang
		FROM area_enriched
		WHERE ($4::text IS NULL OR ten_khu_vuc ILIKE '%' || $4::text || '%')
		  AND ($5::boolean IS NULL OR co_canh_bao = $5::boolean)
		  AND ($6::boolean IS NULL OR co_loi_phan_cung = $6::boolean)
		ORDER BY so_tang ASC, ten_khu_vuc ASC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		buildingID,
		filter.FloorNumber,
		filter.FloorOffset,
		filter.SearchKeyword,
		filter.FireAlertOnly,
		filter.HardwareErrorOnly,
	)
	if err != nil {
		return nil, fmt.Errorf("list area monitoring: %w", err)
	}
	defer rows.Close()

	items := make([]khuvuc.RoomMonitoring, 0)
	for rows.Next() {
		var item khuvuc.RoomMonitoring
		var areaDescription, alertLevel, alertStatus, alertDescription pgtype.Text
		var temperature, smoke pgtype.Float8
		var temperatureUnit, smokeUnit pgtype.Text

		if err := rows.Scan(
			&item.SoTang,
			&item.IDTang,
			&item.IDKhuVuc,
			&item.TenKhuVuc,
			&areaDescription,
			&item.TenLoaiKhuVuc,
			&alertLevel,
			&alertStatus,
			&alertDescription,
			&item.CoCanhBao,
			&item.CoLoiPhanCung,
			&item.TrangThaiPhanCung,
			&temperature,
			&temperatureUnit,
			&smoke,
			&smokeUnit,
			&item.MoTaTinhTrang,
		); err != nil {
			return nil, fmt.Errorf("scan area monitoring row: %w", err)
		}
		if areaDescription.Valid {
			item.MoTaKhuVuc = areaDescription.String
		}
		if alertLevel.Valid {
			item.MucDoCanhBao = alertLevel.String
		}
		if alertStatus.Valid {
			item.TrangThaiCanhBao = alertStatus.String
		}
		if alertDescription.Valid {
			item.MoTaCanhBao = alertDescription.String
		}
		if temperature.Valid {
			value := temperature.Float64
			item.NhietDo = &value
		}
		if temperatureUnit.Valid {
			item.DonViNhietDo = temperatureUnit.String
		}
		if smoke.Valid {
			value := smoke.Float64
			item.NongDoKhoi = &value
		}
		if smokeUnit.Valid {
			item.DonViKhoi = smokeUnit.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate area monitoring rows: %w", err)
	}
	return items, nil
}

func (r *repository) ListFloorMonitoring(
	ctx context.Context,
	buildingID string,
) ([]khuvuc.FloorMonitoring, error) {
	const query = `
		WITH active_fire_floors AS (
			SELECT DISTINCT t.id_tang
			FROM tang t
			JOIN khu_vuc kv ON kv.id_tang = t.id_tang
			JOIN canh_bao cb ON cb.id_khu_vuc = kv.id_khu_vuc
			WHERE t.id_toa_nha = $1::uuid
			  AND cb.trang_thai NOT IN ('RESOLVED', 'FALSE_ALARM', 'CLOSED')
		),
		device_issue_floors AS (
			SELECT DISTINCT t.id_tang
			FROM tang t
			JOIN khu_vuc kv ON kv.id_tang = t.id_tang
			JOIN thiet_bi tb ON tb.id_khu_vuc = kv.id_khu_vuc
			WHERE t.id_toa_nha = $1::uuid
			  AND (
				tb.trang_thai IN ('OFFLINE', 'ERROR', 'FAULT')
				OR tb.lan_cuoi_hoat_dong < NOW() - INTERVAL '2 minutes'
				OR tb.lan_cuoi_hoat_dong IS NULL
			  )
			UNION
			SELECT DISTINCT t.id_tang
			FROM tang t
			JOIN khu_vuc kv ON kv.id_tang = t.id_tang
			JOIN thiet_bi tb ON tb.id_khu_vuc = kv.id_khu_vuc
			JOIN cam_bien cb ON cb.id_thiet_bi = tb.id_thiet_bi
			WHERE t.id_toa_nha = $1::uuid
			  AND cb.trang_thai IN ('OFFLINE', 'ERROR', 'FAULT', 'INACTIVE')
		)
		SELECT
			t.id_tang,
			t.so_tang,
			COUNT(kv.id_khu_vuc)::int AS so_luong_khu_vuc,
			CASE
				WHEN aff.id_tang IS NOT NULL THEN 'RED'
				WHEN dif.id_tang IS NOT NULL THEN 'YELLOW'
				ELSE 'GREEN'
			END AS mau_trang_thai,
			CASE
				WHEN aff.id_tang IS NOT NULL THEN 'FIRE_ALERT'
				WHEN dif.id_tang IS NOT NULL THEN 'DEVICE_ISSUE'
				ELSE 'NORMAL'
			END AS ma_trang_thai
		FROM tang t
		LEFT JOIN khu_vuc kv ON kv.id_tang = t.id_tang
		LEFT JOIN active_fire_floors aff ON aff.id_tang = t.id_tang
		LEFT JOIN device_issue_floors dif ON dif.id_tang = t.id_tang
		WHERE t.id_toa_nha = $1::uuid
		GROUP BY t.id_tang, t.so_tang, aff.id_tang, dif.id_tang
		ORDER BY t.so_tang ASC
	`

	rows, err := r.db.Query(ctx, query, buildingID)
	if err != nil {
		return nil, fmt.Errorf("list floor monitoring: %w", err)
	}
	defer rows.Close()

	items := make([]khuvuc.FloorMonitoring, 0)
	for rows.Next() {
		var item khuvuc.FloorMonitoring
		if err := rows.Scan(
			&item.IDTang,
			&item.SoTang,
			&item.SoLuongKhuVuc,
			&item.MauTrangThai,
			&item.MaTrangThai,
		); err != nil {
			return nil, fmt.Errorf("scan floor monitoring row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate floor monitoring rows: %w", err)
	}
	return items, nil
}

func (r *repository) ListBuildingAreas(
	ctx context.Context,
	buildingID string,
	filter khuvuc.BuildingAreaFilter,
) ([]khuvuc.BuildingArea, int, error) {
	var areaTypeID, statusCode, searchKeyword any
	if filter.AreaTypeID != nil {
		areaTypeID = *filter.AreaTypeID
	}
	if filter.StatusCode != nil {
		statusCode = *filter.StatusCode
	}
	if filter.SearchKeyword != nil {
		searchKeyword = *filter.SearchKeyword
	}

	const listQuery = `
		WITH active_fire_areas AS (
			SELECT DISTINCT cb.id_khu_vuc
			FROM canh_bao cb
			JOIN khu_vuc kv ON kv.id_khu_vuc = cb.id_khu_vuc
			JOIN tang t ON t.id_tang = kv.id_tang
			WHERE t.id_toa_nha = $1
			  AND cb.trang_thai NOT IN ('RESOLVED', 'FALSE_ALARM', 'CLOSED')
		),
		device_issue_areas AS (
			SELECT DISTINCT tb.id_khu_vuc
			FROM thiet_bi tb
			JOIN khu_vuc kv ON kv.id_khu_vuc = tb.id_khu_vuc
			JOIN tang t ON t.id_tang = kv.id_tang
			WHERE t.id_toa_nha = $1
			  AND (
				tb.trang_thai IN ('OFFLINE', 'ERROR', 'FAULT')
				OR tb.lan_cuoi_hoat_dong < NOW() - INTERVAL '15 minutes'
				OR tb.lan_cuoi_hoat_dong IS NULL
			  )
			UNION
			SELECT DISTINCT tb.id_khu_vuc
			FROM cam_bien cam
			JOIN thiet_bi tb ON tb.id_thiet_bi = cam.id_thiet_bi
			JOIN khu_vuc kv ON kv.id_khu_vuc = tb.id_khu_vuc
			JOIN tang t ON t.id_tang = kv.id_tang
			WHERE t.id_toa_nha = $1
			  AND cam.trang_thai IN ('OFFLINE', 'ERROR', 'FAULT', 'INACTIVE')
		),
		area_resident_counts AS (
			SELECT tv.id_khu_vuc,
			       COUNT(tv.id_nguoi_dung)::INT AS so_luong_cu_dan
			FROM thanh_vien_khu_vuc tv
			JOIN khu_vuc kv ON kv.id_khu_vuc = tv.id_khu_vuc
			JOIN tang t ON t.id_tang = kv.id_tang
			WHERE t.id_toa_nha = $1
			  AND tv.trang_thai = 'DANG_HOAT_DONG'
			GROUP BY tv.id_khu_vuc
		),
		evaluated_areas AS (
			SELECT kv.id_khu_vuc,
			       kv.ten_khu_vuc,
			       kv.mo_ta AS mo_ta_khu_vuc,
			       t.id_tang,
			       t.so_tang,
			       lk.id_loai_khu_vuc,
			       lk.ten_loai_khu_vuc,
			       COALESCE(arc.so_luong_cu_dan, 0) AS so_luong_cu_dan,
			       CASE
			         WHEN afa.id_khu_vuc IS NOT NULL THEN 'RED'
			         WHEN dia.id_khu_vuc IS NOT NULL THEN 'YELLOW'
			         ELSE 'GREEN'
			       END AS mau_trang_thai,
			       CASE
			         WHEN afa.id_khu_vuc IS NOT NULL THEN 'FIRE_ALERT'
			         WHEN dia.id_khu_vuc IS NOT NULL THEN 'DEVICE_ISSUE'
			         ELSE 'NORMAL'
			       END AS ma_trang_thai
			FROM khu_vuc kv
			JOIN tang t ON t.id_tang = kv.id_tang
			JOIN loai_khu_vuc lk ON lk.id_loai_khu_vuc = kv.id_loai_khu_vuc
			LEFT JOIN active_fire_areas afa ON afa.id_khu_vuc = kv.id_khu_vuc
			LEFT JOIN device_issue_areas dia ON dia.id_khu_vuc = kv.id_khu_vuc
			LEFT JOIN area_resident_counts arc ON arc.id_khu_vuc = kv.id_khu_vuc
			WHERE t.id_toa_nha = $1
			  AND ($2::UUID IS NULL OR kv.id_loai_khu_vuc = $2)
			  AND ($3::INT IS NULL OR t.so_tang = $3)
			  AND ($4::TEXT IS NULL OR kv.ten_khu_vuc ILIKE '%' || $4 || '%')
		)
		SELECT id_khu_vuc, ten_khu_vuc, mo_ta_khu_vuc, id_tang, so_tang,
		       id_loai_khu_vuc, ten_loai_khu_vuc, so_luong_cu_dan,
		       mau_trang_thai, ma_trang_thai
		FROM evaluated_areas
		WHERE ($5::TEXT IS NULL OR ma_trang_thai = $5)
		ORDER BY so_tang ASC, ten_khu_vuc ASC
		LIMIT $6 OFFSET $7
	`

	rows, err := r.db.Query(ctx, listQuery, buildingID, areaTypeID, filter.FloorNumber,
		searchKeyword, statusCode, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list building areas: %w", err)
	}
	defer rows.Close()

	items := make([]khuvuc.BuildingArea, 0, filter.Limit)
	for rows.Next() {
		var item khuvuc.BuildingArea
		var description pgtype.Text
		if err := rows.Scan(
			&item.IDKhuVuc, &item.TenKhuVuc, &description, &item.IDTang,
			&item.SoTang, &item.IDLoaiKhuVuc, &item.TenLoaiKhuVuc,
			&item.SoLuongCuDan, &item.MauTrangThai, &item.MaTrangThai,
		); err != nil {
			return nil, 0, fmt.Errorf("scan building area row: %w", err)
		}
		if description.Valid {
			item.MoTaKhuVuc = description.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate building area rows: %w", err)
	}

	const countQuery = `
		WITH active_fire_areas AS (
			SELECT DISTINCT cb.id_khu_vuc
			FROM canh_bao cb
			JOIN khu_vuc kv ON kv.id_khu_vuc = cb.id_khu_vuc
			JOIN tang t ON t.id_tang = kv.id_tang
			WHERE t.id_toa_nha = $1
			  AND cb.trang_thai NOT IN ('RESOLVED', 'FALSE_ALARM', 'CLOSED')
		),
		device_issue_areas AS (
			SELECT DISTINCT tb.id_khu_vuc
			FROM thiet_bi tb
			JOIN khu_vuc kv ON kv.id_khu_vuc = tb.id_khu_vuc
			JOIN tang t ON t.id_tang = kv.id_tang
			WHERE t.id_toa_nha = $1
			  AND (tb.trang_thai IN ('OFFLINE', 'ERROR', 'FAULT')
			       OR tb.lan_cuoi_hoat_dong < NOW() - INTERVAL '15 minutes'
			       OR tb.lan_cuoi_hoat_dong IS NULL)
			UNION
			SELECT DISTINCT tb.id_khu_vuc
			FROM cam_bien cam
			JOIN thiet_bi tb ON tb.id_thiet_bi = cam.id_thiet_bi
			JOIN khu_vuc kv ON kv.id_khu_vuc = tb.id_khu_vuc
			JOIN tang t ON t.id_tang = kv.id_tang
			WHERE t.id_toa_nha = $1
			  AND cam.trang_thai IN ('OFFLINE', 'ERROR', 'FAULT', 'INACTIVE')
		),
		evaluated_areas AS (
			SELECT kv.id_khu_vuc,
			       CASE
			         WHEN afa.id_khu_vuc IS NOT NULL THEN 'FIRE_ALERT'
			         WHEN dia.id_khu_vuc IS NOT NULL THEN 'DEVICE_ISSUE'
			         ELSE 'NORMAL'
			       END AS ma_trang_thai
			FROM khu_vuc kv
			JOIN tang t ON t.id_tang = kv.id_tang
			LEFT JOIN active_fire_areas afa ON afa.id_khu_vuc = kv.id_khu_vuc
			LEFT JOIN device_issue_areas dia ON dia.id_khu_vuc = kv.id_khu_vuc
			WHERE t.id_toa_nha = $1
			  AND ($2::UUID IS NULL OR kv.id_loai_khu_vuc = $2)
			  AND ($3::INT IS NULL OR t.so_tang = $3)
			  AND ($4::TEXT IS NULL OR kv.ten_khu_vuc ILIKE '%' || $4 || '%')
		)
		SELECT COUNT(*)::INT
		FROM evaluated_areas
		WHERE ($5::TEXT IS NULL OR ma_trang_thai = $5)
	`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, buildingID, areaTypeID, filter.FloorNumber,
		searchKeyword, statusCode).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count building areas: %w", err)
	}
	return items, total, nil
}

func (r *repository) ListBuildingFloors(
	ctx context.Context,
	buildingID string,
) ([]khuvuc.BuildingFloor, error) {
	const query = `
		SELECT t.id_tang, t.so_tang
		FROM tang t
		WHERE t.id_toa_nha = $1
		ORDER BY t.so_tang DESC
	`
	rows, err := r.db.Query(ctx, query, buildingID)
	if err != nil {
		return nil, fmt.Errorf("list building floors: %w", err)
	}
	defer rows.Close()
	items := make([]khuvuc.BuildingFloor, 0)
	for rows.Next() {
		var item khuvuc.BuildingFloor
		if err := rows.Scan(&item.IDTang, &item.SoTang); err != nil {
			return nil, fmt.Errorf("scan building floor row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate building floor rows: %w", err)
	}
	return items, nil
}

func (r *repository) ListBuildingAreaTypes(
	ctx context.Context,
	buildingID string,
) ([]khuvuc.BuildingAreaType, error) {
	const query = `
		SELECT lkv.id_loai_khu_vuc, lkv.ten_loai_khu_vuc
		FROM loai_khu_vuc lkv
		JOIN toa_nha tn ON tn.id_loai_mo_hinh = lkv.id_loai_mo_hinh
		WHERE tn.id_toa_nha = $1
		ORDER BY lkv.ten_loai_khu_vuc ASC
	`
	rows, err := r.db.Query(ctx, query, buildingID)
	if err != nil {
		return nil, fmt.Errorf("list building area types: %w", err)
	}
	defer rows.Close()
	items := make([]khuvuc.BuildingAreaType, 0)
	for rows.Next() {
		var item khuvuc.BuildingAreaType
		if err := rows.Scan(&item.IDLoaiKhuVuc, &item.TenLoaiKhuVuc); err != nil {
			return nil, fmt.Errorf("scan building area type row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate building area type rows: %w", err)
	}
	return items, nil
}
