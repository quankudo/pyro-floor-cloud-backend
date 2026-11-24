package phanconggiamsat

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/database"
)

var (
	ErrTargetNotFound = apperror.New(
		http.StatusNotFound,
		apperror.AppErrorNotFound,
		"building, floor, or area was not found in the requested building",
		nil,
	)
	ErrAlreadyAssigned = apperror.New(
		http.StatusConflict,
		apperror.AppErrorConflict,
		"user already has an active assignment for this scope",
		nil,
	)
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("supervisor assignment repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) AddBatch(
	ctx context.Context,
	tx database.DBTX,
	input AddAssignmentsInput,
) error {
	if _, err := uuid.Parse(input.BuildingID); err != nil {
		return invalidTargetID("id_toa_nha")
	}
	if len(input.Assignments) == 0 {
		return nil
	}
	for _, assignment := range input.Assignments {
		if _, err := uuid.Parse(assignment.UserID); err != nil {
			return invalidTargetID("id_nguoi_dung")
		}
		if assignment.FloorID != "" {
			if _, err := uuid.Parse(assignment.FloorID); err != nil {
				return invalidTargetID("id_tang")
			}
		}
		if assignment.AreaID != "" {
			if _, err := uuid.Parse(assignment.AreaID); err != nil {
				return invalidTargetID("id_khu_vuc")
			}
		}
		if assignment.FloorID == "" && assignment.AreaID != "" {
			return apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"area assignment requires a floor ID",
				nil,
			)
		}
	}
	if tx == nil {
		return errors.New("database transaction is required")
	}
	return s.repository.AddBatch(ctx, tx, input)
}

func (s *Service) ListAssignmentScopes(
	ctx context.Context,
	buildingID string,
	filter AssignmentScopeFilter,
) ([]AssignmentScope, error) {
	buildingID = strings.TrimSpace(buildingID)
	if _, err := uuid.Parse(buildingID); err != nil {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}
	if filter.UserID != nil {
		value := strings.TrimSpace(*filter.UserID)
		if value == "" {
			filter.UserID = nil
		} else {
			if _, err := uuid.Parse(value); err != nil {
				return nil, invalidTargetID("id_nguoi_dung")
			}
			filter.UserID = &value
		}
	}
	if filter.AreaName != nil {
		value := strings.TrimSpace(*filter.AreaName)
		if value == "" {
			filter.AreaName = nil
		} else {
			filter.AreaName = &value
		}
	}
	if filter.AssignmentStatus != nil {
		value := strings.ToUpper(strings.TrimSpace(*filter.AssignmentStatus))
		switch value {
		case "DA_PHAN_CONG", "CHUA_PHAN_CONG":
			filter.AssignmentStatus = &value
		default:
			return nil, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"trang_thai_phan_cong must be DA_PHAN_CONG or CHUA_PHAN_CONG",
				map[string]any{"field": "trang_thai_phan_cong"},
			)
		}
	}

	items, err := s.repository.ListAssignmentScopes(ctx, buildingID, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []AssignmentScope{}
	}
	return items, nil
}

func invalidTargetID(field string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		"assignment target ID must be a UUID",
		map[string]any{"field": field},
	)
}
