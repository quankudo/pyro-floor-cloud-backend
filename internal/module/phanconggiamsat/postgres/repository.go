package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/database"
	"github.com/pyrofloor/cloud-backend/internal/module/phanconggiamsat"
)

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(databases ...*pgxpool.Pool) *repository {
	if len(databases) == 0 {
		return &repository{}
	}
	return &repository{db: databases[0]}
}

func (r *repository) AddBatch(
	ctx context.Context,
	tx database.DBTX,
	input phanconggiamsat.AddAssignmentsInput,
) error {
	const lockQuery = `
		SELECT id_toa_nha::text
		FROM toa_nha
		WHERE id_toa_nha = $1::uuid
		FOR UPDATE
	`
	var lockedBuildingID string
	if err := tx.QueryRow(ctx, lockQuery, input.BuildingID).Scan(&lockedBuildingID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return phanconggiamsat.ErrTargetNotFound
		}
		return fmt.Errorf("lock building for supervisor assignment batch: %w", err)
	}

	userIDs := make([]string, len(input.Assignments))
	floorIDs := make([]string, len(input.Assignments))
	areaIDs := make([]string, len(input.Assignments))
	assignmentIDs := make([]string, len(input.Assignments))
	for i, assignment := range input.Assignments {
		userIDs[i] = assignment.UserID
		floorIDs[i] = assignment.FloorID
		areaIDs[i] = assignment.AreaID
		assignmentIDs[i] = uuid.NewString()
	}

	const validateScopesQuery = `
		SELECT COUNT(*)
		FROM unnest($1::text[], $2::text[], $3::text[])
			AS requested(user_id, floor_id, area_id)
		WHERE (NULLIF(requested.floor_id, '') IS NULL OR EXISTS (
			SELECT 1
			FROM tang AS floor
			WHERE floor.id_tang = NULLIF(requested.floor_id, '')::uuid
				AND floor.id_toa_nha = $4::uuid
		))
		AND (NULLIF(requested.area_id, '') IS NULL OR EXISTS (
			SELECT 1
			FROM khu_vuc AS area
			INNER JOIN tang AS floor
				ON floor.id_tang = area.id_tang
			WHERE area.id_khu_vuc = NULLIF(requested.area_id, '')::uuid
				AND floor.id_toa_nha = $4::uuid
				AND floor.id_tang = NULLIF(requested.floor_id, '')::uuid
		))
	`
	var validScopeCount int
	if err := tx.QueryRow(
		ctx,
		validateScopesQuery,
		userIDs,
		floorIDs,
		areaIDs,
		input.BuildingID,
	).Scan(&validScopeCount); err != nil {
		return fmt.Errorf("validate supervisor assignment scopes in building: %w", err)
	}
	if validScopeCount != len(input.Assignments) {
		return phanconggiamsat.ErrTargetNotFound
	}

	const duplicateQuery = `
		SELECT EXISTS (
			SELECT 1
			FROM unnest($1::text[], $2::text[], $3::text[])
				AS requested(user_id, floor_id, area_id)
			INNER JOIN phan_cong_giam_sat AS assignment
				ON assignment.id_nguoi_dung = requested.user_id::uuid
				AND assignment.id_toa_nha = $4::uuid
				AND assignment.id_tang IS NOT DISTINCT FROM NULLIF(requested.floor_id, '')::uuid
				AND assignment.id_khu_vuc IS NOT DISTINCT FROM NULLIF(requested.area_id, '')::uuid
			WHERE UPPER(assignment.trang_thai) IN ($5, $6)
		)
	`
	var hasDuplicate bool
	if err := tx.QueryRow(
		ctx,
		duplicateQuery,
		userIDs,
		floorIDs,
		areaIDs,
		input.BuildingID,
		phanconggiamsat.StatusActive,
		phanconggiamsat.StatusLegacyActive,
	).Scan(&hasDuplicate); err != nil {
		return fmt.Errorf("check active supervisor assignment batch: %w", err)
	}
	if hasDuplicate {
		return phanconggiamsat.ErrAlreadyAssigned
	}

	const insertBatchQuery = `
		INSERT INTO phan_cong_giam_sat (
			id_phan_cong_giam_sat,
			id_nguoi_dung,
			id_toa_nha,
			id_tang,
			id_khu_vuc,
			loai_phan_cong,
			trang_thai
		)
		SELECT
			requested.assignment_id::uuid,
			requested.user_id::uuid,
			$5::uuid,
			NULLIF(requested.floor_id, '')::uuid,
			NULLIF(requested.area_id, '')::uuid,
			CASE
				WHEN NULLIF(requested.area_id, '') IS NOT NULL THEN 'KHU_VUC'
				WHEN NULLIF(requested.floor_id, '') IS NOT NULL THEN 'TANG'
				ELSE 'TOA_NHA'
			END,
			$6
		FROM unnest(
			$1::text[],
			$2::text[],
			$3::text[],
			$4::text[]
		) AS requested(assignment_id, user_id, floor_id, area_id)
	`
	if _, err := tx.Exec(
		ctx,
		insertBatchQuery,
		assignmentIDs,
		userIDs,
		floorIDs,
		areaIDs,
		input.BuildingID,
		phanconggiamsat.StatusActive,
	); err != nil {
		return fmt.Errorf("create supervisor assignment batch: %w", err)
	}
	return nil
}

func (r *repository) ListAssignmentScopes(
	ctx context.Context,
	buildingID string,
	filter phanconggiamsat.AssignmentScopeFilter,
) ([]phanconggiamsat.AssignmentScope, error) {
	if r.db == nil {
		return nil, errors.New("database pool is required")
	}
	var floorNumber, areaName, userID, status any
	if filter.FloorNumber != nil {
		floorNumber = *filter.FloorNumber
	}
	if filter.AreaName != nil {
		areaName = *filter.AreaName
	}
	if filter.UserID != nil {
		userID = *filter.UserID
	}
	if filter.AssignmentStatus != nil {
		status = *filter.AssignmentStatus
	}

	const query = `
		WITH base_scopes AS (
		    SELECT t.id_tang, t.so_tang, kv.id_khu_vuc, kv.ten_khu_vuc, t.id_toa_nha
		    FROM tang t
		    LEFT JOIN khu_vuc kv ON kv.id_tang = t.id_tang
		    WHERE t.id_toa_nha = $1
		      AND ($2::INT IS NULL OR t.so_tang = $2)
		      AND ($3::TEXT IS NULL OR kv.ten_khu_vuc ILIKE '%' || $3 || '%')
		),
		effective_assignments AS (
		    SELECT bs.id_tang, bs.so_tang, bs.id_khu_vuc, bs.ten_khu_vuc,
		           pc.id_nguoi_dung, pc.trang_thai AS trang_thai_raw
		    FROM base_scopes bs
		    JOIN phan_cong_giam_sat pc
		      ON bs.id_khu_vuc IS NOT NULL
		     AND pc.id_khu_vuc = bs.id_khu_vuc
		     AND pc.id_toa_nha = bs.id_toa_nha
		     AND (pc.loai_phan_cong = 'KHU_VUC' OR pc.loai_phan_cong IS NULL)

		    UNION ALL

		    SELECT bs.id_tang, bs.so_tang, bs.id_khu_vuc, bs.ten_khu_vuc,
		           pc.id_nguoi_dung, pc.trang_thai AS trang_thai_raw
		    FROM base_scopes bs
		    JOIN phan_cong_giam_sat pc
		      ON pc.id_tang = bs.id_tang
		     AND pc.id_khu_vuc IS NULL
		     AND pc.id_toa_nha = bs.id_toa_nha
		     AND (pc.loai_phan_cong = 'TANG' OR pc.loai_phan_cong IS NULL)
		    WHERE NOT EXISTS (
		        SELECT 1 FROM phan_cong_giam_sat pck
		        WHERE pck.id_khu_vuc = bs.id_khu_vuc
		          AND pck.id_toa_nha = bs.id_toa_nha
		    )

		    UNION ALL

		    SELECT bs.id_tang, bs.so_tang, bs.id_khu_vuc, bs.ten_khu_vuc,
		           pc.id_nguoi_dung, pc.trang_thai AS trang_thai_raw
		    FROM base_scopes bs
		    JOIN phan_cong_giam_sat pc
		      ON pc.id_toa_nha = bs.id_toa_nha
		     AND pc.id_tang IS NULL
		     AND pc.id_khu_vuc IS NULL
		     AND (pc.loai_phan_cong = 'TOA_NHA' OR pc.loai_phan_cong IS NULL)
		),
		normalized_rows AS (
		    SELECT DISTINCT
		        ea.id_tang, ea.so_tang, ea.id_khu_vuc, ea.ten_khu_vuc,
		        CASE
		            WHEN UPPER(ea.trang_thai_raw) IN ('ACTIVE', 'DA_PHAN_CONG')
		            THEN ea.id_nguoi_dung ELSE NULL
		        END AS id_nguoi_dung,
		        CASE
		            WHEN UPPER(ea.trang_thai_raw) IN ('ACTIVE', 'DA_PHAN_CONG')
		            THEN 'DA_PHAN_CONG' ELSE 'CHUA_PHAN_CONG'
		        END AS trang_thai_phan_cong
		    FROM effective_assignments ea
		),
		cleaned_rows AS (
		    SELECT nr.*
		    FROM normalized_rows nr
		    WHERE nr.trang_thai_phan_cong = 'DA_PHAN_CONG'
		       OR NOT EXISTS (
		           SELECT 1 FROM normalized_rows nr2
		           WHERE nr2.id_tang = nr.id_tang
		             AND nr2.id_khu_vuc IS NOT DISTINCT FROM nr.id_khu_vuc
		             AND nr2.trang_thai_phan_cong = 'DA_PHAN_CONG'
		       )
		),
		unassigned_scopes AS (
		    SELECT DISTINCT bs.id_tang, bs.so_tang, bs.id_khu_vuc, bs.ten_khu_vuc,
		           NULL::UUID AS id_nguoi_dung, 'CHUA_PHAN_CONG'::TEXT AS trang_thai_phan_cong
		    FROM base_scopes bs
		    WHERE NOT EXISTS (
		        SELECT 1 FROM cleaned_rows cr
		        WHERE cr.id_tang = bs.id_tang
		          AND cr.id_khu_vuc IS NOT DISTINCT FROM bs.id_khu_vuc
		    )
		),
		final_rows AS (
		    SELECT * FROM cleaned_rows
		    UNION ALL
		    SELECT * FROM unassigned_scopes
		)
		SELECT f.id_tang, f.so_tang, f.id_khu_vuc, f.ten_khu_vuc,
		       f.id_nguoi_dung, nd.ten_nguoi_dung, f.trang_thai_phan_cong
		FROM final_rows f
		LEFT JOIN nguoi_dung nd
		  ON f.trang_thai_phan_cong = 'DA_PHAN_CONG'
		 AND nd.id_nguoi_dung = f.id_nguoi_dung
		WHERE ($4::UUID IS NULL OR f.id_nguoi_dung = $4)
		  AND ($5::TEXT IS NULL OR f.trang_thai_phan_cong = $5)
		ORDER BY
		    f.so_tang DESC,
		    f.ten_khu_vuc ASC NULLS LAST,
		    nd.ten_nguoi_dung ASC NULLS LAST
	`
	rows, err := r.db.Query(ctx, query, buildingID, floorNumber, areaName, userID, status)
	if err != nil {
		return nil, fmt.Errorf("list assignment scopes: %w", err)
	}
	defer rows.Close()

	items := make([]phanconggiamsat.AssignmentScope, 0)
	for rows.Next() {
		var item phanconggiamsat.AssignmentScope
		var areaID, areaName, userID, userName pgtype.Text
		if err := rows.Scan(
			&item.IDTang,
			&item.SoTang,
			&areaID,
			&areaName,
			&userID,
			&userName,
			&item.TrangThaiPhanCong,
		); err != nil {
			return nil, fmt.Errorf("scan assignment scope row: %w", err)
		}
		if areaID.Valid {
			value := areaID.String
			item.IDKhuVuc = &value
		}
		if areaName.Valid {
			value := areaName.String
			item.TenKhuVuc = &value
		}
		if userID.Valid {
			value := userID.String
			item.IDNguoiDung = &value
		}
		if userName.Valid {
			value := userName.String
			item.TenNguoiDung = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignment scope rows: %w", err)
	}
	return items, nil
}
