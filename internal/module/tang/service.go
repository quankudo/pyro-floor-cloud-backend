package tang

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("floor repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) ListByBuildingID(
	ctx context.Context,
	buildingID string,
) ([]FloorsResponse, error) {
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"building ID must be a valid UUID",
			map[string]any{"field": "id"},
		)
	}

	items, err := s.repository.ListByBuildingID(ctx, buildingID)
	if err != nil {
		return nil, err
	}
	result := make([]FloorsResponse, len(items))
	for i, item := range items {
		result[i] = FloorsResponse{
			IDTang: item.IDTang,
			SoTang: item.SoTang,
		}
	}
	return result, nil
}

func (s *Service) ListForBuilding(ctx context.Context, buildingID string) ([]Tang, error) {
	buildingID = strings.TrimSpace(buildingID)
	if !isUUID(buildingID) {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"building ID must be a valid UUID",
			map[string]any{"field": "id"},
		)
	}
	items, err := s.repository.ListForBuilding(ctx, buildingID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []Tang{}
	}
	return items, nil
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
