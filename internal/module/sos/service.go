package sos

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
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
		return nil, errors.New("SOS repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) ListCompleted(
	ctx context.Context,
	buildingID string,
	filter ListFilter,
) (*Page, error) {
	buildingID = strings.TrimSpace(buildingID)
	if _, err := uuid.Parse(buildingID); err != nil {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxPageSize {
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
	if filter.FromTime != nil && filter.ToTime != nil && filter.FromTime.After(*filter.ToTime) {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"from_time must not be after to_time",
			map[string]any{"field": "from_time"},
		)
	}
	normalizeFilterText(&filter.LoaiSOS)
	normalizeFilterText(&filter.Keyword)

	items, total, err := s.repository.ListCompleted(ctx, buildingID, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []SOSReport{}
	}
	return &Page{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (s *Service) ListActive(
	ctx context.Context,
	buildingID string,
	userID string,
	filter ActiveFilter,
) (*ActivePage, error) {
	buildingID = strings.TrimSpace(buildingID)
	userID = strings.TrimSpace(userID)
	if _, err := uuid.Parse(buildingID); err != nil {
		return nil, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		)
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, apperror.New(
			http.StatusUnauthorized,
			apperror.AppErrorUnauthenticated,
			"user identity is missing from authentication token",
			nil,
		)
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxPageSize {
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
	normalizeFilterText(&filter.LoaiSOS)
	normalizeFilterText(&filter.TrangThai)
	if filter.TrangThai != nil && *filter.TrangThai != "NEW" && *filter.TrangThai != "ACKNOWLEDGED" {
		return nil, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"trang_thai must be NEW or ACKNOWLEDGED",
			map[string]any{"field": "trang_thai"},
		)
	}

	items, total, err := s.repository.ListActive(ctx, buildingID, userID, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ActiveSOSReport{}
	}
	return &ActivePage{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func normalizeFilterText(value **string) {
	if *value == nil {
		return
	}
	trimmed := strings.TrimSpace(**value)
	if trimmed == "" {
		*value = nil
		return
	}
	*value = &trimmed
}
