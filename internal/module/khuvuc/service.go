package khuvuc

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

type Service struct {
	repository Repository
}

const (
	DefaultBuildingAreaPageSize = 20
	MaxBuildingAreaPageSize     = 100
)

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("area repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) ListByFloorID(ctx context.Context, floorID string) ([]KhuVuc, error) {
	floorID = strings.TrimSpace(floorID)
	if !isUUID(floorID) {
		return nil, invalidFloorIDError()
	}

	items, err := s.repository.ListByFloorID(ctx, floorID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []KhuVuc{}
	}
	return items, nil
}

func (s *Service) ListShortByFloorID(ctx context.Context, floorID string) ([]KhuVucShort, error) {
	floorID = strings.TrimSpace(floorID)
	if !isUUID(floorID) {
		return nil, invalidFloorIDError()
	}

	items, err := s.repository.ListShortByFloorID(ctx, floorID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []KhuVucShort{}
	}
	return items, nil
}

func (s *Service) ListMonitoring(
	ctx context.Context,
	buildingID string,
	filter MonitoringFilter,
) ([]RoomMonitoring, error) {
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}
	if filter.FloorOffset < 0 {
		filter.FloorOffset = 0
	}

	items, err := s.repository.ListMonitoring(ctx, buildingID, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []RoomMonitoring{}
	}
	return items, nil
}

func (s *Service) ListFloorMonitoring(
	ctx context.Context,
	buildingID string,
) ([]FloorMonitoring, error) {
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}

	items, err := s.repository.ListFloorMonitoring(ctx, buildingID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []FloorMonitoring{}
	}
	return items, nil
}

func (s *Service) ListBuildingAreas(
	ctx context.Context,
	buildingID string,
	filter BuildingAreaFilter,
) (*BuildingAreaPage, error) {
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultBuildingAreaPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxBuildingAreaPageSize {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"limit must be between 1 and 100",
			map[string]any{"field": "limit"},
		)
	}
	if filter.Offset < 0 {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"offset must not be negative",
			map[string]any{"field": "offset"},
		)
	}
	if filter.AreaTypeID != nil {
		value := strings.TrimSpace(*filter.AreaTypeID)
		if value == "" || !isUUID(value) {
			return nil, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"area type ID must be a valid UUID",
				map[string]any{"field": "id_loai_khu_vuc"},
			)
		}
		filter.AreaTypeID = &value
	}
	if filter.StatusCode != nil {
		value := strings.TrimSpace(*filter.StatusCode)
		switch value {
		case "FIRE_ALERT", "DEVICE_ISSUE", "NORMAL":
			filter.StatusCode = &value
		default:
			return nil, apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"ma_trang_thai must be one of FIRE_ALERT, DEVICE_ISSUE, NORMAL",
				map[string]any{"field": "ma_trang_thai"},
			)
		}
	}
	if filter.SearchKeyword != nil {
		value := strings.TrimSpace(*filter.SearchKeyword)
		if value == "" {
			filter.SearchKeyword = nil
		} else {
			filter.SearchKeyword = &value
		}
	}

	items, total, err := s.repository.ListBuildingAreas(ctx, buildingID, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []BuildingArea{}
	}

	page := &BuildingAreaPage{
		Items:      items,
		TotalCount: total,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		Floors:     []BuildingFloor{},
		AreaTypes:  []BuildingAreaType{},
	}

	hasAreaFilters := filter.AreaTypeID != nil ||
		filter.FloorNumber != nil ||
		filter.StatusCode != nil ||
		filter.SearchKeyword != nil
	if hasAreaFilters {
		return page, nil
	}

	page.Floors, err = s.repository.ListBuildingFloors(ctx, buildingID)
	if err != nil {
		return nil, err
	}
	if page.Floors == nil {
		page.Floors = []BuildingFloor{}
	}
	page.AreaTypes, err = s.repository.ListBuildingAreaTypes(ctx, buildingID)
	if err != nil {
		return nil, err
	}
	if page.AreaTypes == nil {
		page.AreaTypes = []BuildingAreaType{}
	}
	return page, nil
}

func invalidFloorIDError() error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		"floor ID must be a valid UUID",
		map[string]any{"field": "id"},
	)
}

func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return false
			}
		default:
			if !((char >= '0' && char <= '9') ||
				(char >= 'a' && char <= 'f') ||
				(char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}
