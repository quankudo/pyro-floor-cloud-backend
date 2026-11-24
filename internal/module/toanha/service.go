package toanha

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/database"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type toaNhaService struct {
	toaNhaRepo ToaNhaRepository
}

func NewToaNhaService(toaNhaRepo ToaNhaRepository) *toaNhaService {
	return &toaNhaService{
		toaNhaRepo: toaNhaRepo,
	}
}

func (s *toaNhaService) FindByID(
	ctx context.Context,
	id string,
) (*ToaNha, error) {
	return s.toaNhaRepo.FindByID(ctx, id)
}

func (s *toaNhaService) EnsureOwner(
	ctx context.Context,
	tx database.DBTX,
	buildingID, ownerID string,
) error {
	if !isUUID(buildingID) {
		return invalidFilter("id_toa_nha", "id_toa_nha must be a valid UUID")
	}
	if !isUUID(ownerID) {
		return apperror.New(
			http.StatusUnauthorized,
			apperror.AppErrorUnauthenticated,
			"authenticated owner is required",
			nil,
		)
	}
	if tx == nil {
		return errors.New("database transaction is required")
	}

	buildingOwnerID, err := s.toaNhaRepo.FindOwnerID(ctx, tx, buildingID)
	if err != nil {
		if errors.Is(err, ErrToaNhaNotFound) {
			return apperror.New(
				http.StatusNotFound,
				apperror.AppErrorNotFound,
				"building was not found",
				nil,
			)
		}
		return err
	}
	if !strings.EqualFold(buildingOwnerID, ownerID) {
		return ErrBuildingForbidden
	}
	return nil
}

func (s *toaNhaService) List(
	ctx context.Context,
	filter ListFilter,
) (*Page, error) {
	filter.TrangThai = strings.TrimSpace(filter.TrangThai)
	filter.IDLoaiMoHinh = strings.TrimSpace(filter.IDLoaiMoHinh)
	filter.TenToaNha = strings.TrimSpace(filter.TenToaNha)

	if filter.Limit == 0 {
		filter.Limit = DefaultPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxPageSize {
		return nil, invalidFilter("limit", "limit must be between 1 and 100")
	}
	if filter.Offset < 0 {
		return nil, invalidFilter("offset", "offset must not be negative")
	}
	if utf8.RuneCountInString(filter.TrangThai) > 20 {
		return nil, invalidFilter("trang_thai", "trang_thai must be at most 20 characters")
	}
	if filter.IDLoaiMoHinh != "" && !isUUID(filter.IDLoaiMoHinh) {
		return nil, invalidFilter("id_loai_mo_hinh", "id_loai_mo_hinh must be a valid UUID")
	}
	if !utf8.ValidString(filter.TenToaNha) ||
		utf8.RuneCountInString(filter.TenToaNha) > 150 {
		return nil, invalidFilter("ten_toa_nha", "ten_toa_nha must be at most 150 characters")
	}

	items, total, err := s.toaNhaRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ToaNha{}
	}

	return &Page{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func invalidFilter(field, message string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		message,
		map[string]any{"field": field},
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
