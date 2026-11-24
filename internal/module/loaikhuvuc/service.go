package loaikhuvuc

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

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("area type repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) ListByModelTypeID(
	ctx context.Context,
	modelTypeID string,
) ([]LoaiKhuVuc, error) {
	modelTypeID = strings.TrimSpace(modelTypeID)
	if !isUUID(modelTypeID) {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"model type ID must be a valid UUID",
			map[string]any{"field": "id"},
		)
	}

	items, err := s.repository.ListByModelTypeID(ctx, modelTypeID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []LoaiKhuVuc{}
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
