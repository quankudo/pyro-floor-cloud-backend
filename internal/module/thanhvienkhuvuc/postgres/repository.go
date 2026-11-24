package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pyrofloor/cloud-backend/internal/database"
	"github.com/pyrofloor/cloud-backend/internal/module/thanhvienkhuvuc"
)

type repository struct{}

func NewRepository() *repository {
	return &repository{}
}

func (r *repository) AddBatch(
	ctx context.Context,
	tx database.DBTX,
	input thanhvienkhuvuc.AddMembersInput,
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
			return thanhvienkhuvuc.ErrTargetNotFound
		}
		return fmt.Errorf("lock building for area membership batch: %w", err)
	}

	userIDs := make([]string, len(input.Members))
	areaIDs := make([]string, len(input.Members))
	for i, member := range input.Members {
		userIDs[i] = member.UserID
		areaIDs[i] = member.AreaID
	}

	const validateAreasQuery = `
		SELECT COUNT(*)
		FROM unnest($1::text[]) AS requested(area_id)
		INNER JOIN khu_vuc AS area
			ON area.id_khu_vuc = requested.area_id::uuid
		INNER JOIN tang AS floor
			ON floor.id_tang = area.id_tang
		WHERE floor.id_toa_nha = $2::uuid
	`
	var validAreaCount int
	if err := tx.QueryRow(ctx, validateAreasQuery, areaIDs, input.BuildingID).Scan(&validAreaCount); err != nil {
		return fmt.Errorf("validate resident areas in building: %w", err)
	}
	if validAreaCount != len(input.Members) {
		return thanhvienkhuvuc.ErrTargetNotFound
	}

	const insertBatchQuery = `
		WITH requested AS (
			SELECT user_id, area_id
			FROM unnest($1::text[], $2::text[]) AS requested(user_id, area_id)
		),
		inserted AS (
			INSERT INTO thanh_vien_khu_vuc (
				id_nguoi_dung,
				id_khu_vuc,
				trang_thai
			)
			SELECT user_id::uuid, area_id::uuid, $3
			FROM requested
			ON CONFLICT (id_nguoi_dung, id_khu_vuc) DO UPDATE
				SET trang_thai = EXCLUDED.trang_thai
				WHERE UPPER(thanh_vien_khu_vuc.trang_thai) = 'INACTIVE'
			RETURNING 1
		)
		SELECT COUNT(*) FROM inserted
	`
	var insertedCount int
	if err := tx.QueryRow(
		ctx,
		insertBatchQuery,
		userIDs,
		areaIDs,
		thanhvienkhuvuc.StatusActive,
	).Scan(&insertedCount); err != nil {
		return fmt.Errorf("create area membership batch: %w", err)
	}
	if insertedCount != len(input.Members) {
		return thanhvienkhuvuc.ErrAlreadyMember
	}
	return nil
}
