package phanconggiamsat

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/authctx"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

type AssignmentUseCase interface {
	ListAssignmentScopes(ctx context.Context, buildingID string, filter AssignmentScopeFilter) ([]AssignmentScope, error)
}

type Handler struct {
	service AssignmentUseCase
}

type assignmentScopeResponse struct {
	IDTang            string  `json:"id_tang"`
	SoTang            int     `json:"so_tang"`
	IDKhuVuc          *string `json:"id_khu_vuc"`
	TenKhuVuc         *string `json:"ten_khu_vuc"`
	IDNguoiDung       *string `json:"id_nguoi_dung"`
	TenNguoiDung      *string `json:"ten_nguoi_dung"`
	TrangThaiPhanCong string  `json:"trang_thai_phan_cong"`
}

func NewHandler(service AssignmentUseCase) (*Handler, error) {
	if service == nil {
		return nil, errors.New("supervisor assignment service is required")
	}
	return &Handler{service: service}, nil
}

func (h *Handler) ListAssignmentScopes(w http.ResponseWriter, r *http.Request) {
	principal, ok := authctx.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.IDToaNha) == "" {
		httpx.WriteError(w, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		))
		return
	}

	filter, err := parseAssignmentScopeFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	items, err := h.service.ListAssignmentScopes(r.Context(), principal.IDToaNha, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]assignmentScopeResponse, len(items))
	for i, item := range items {
		response[i] = assignmentScopeResponse{
			IDTang:            item.IDTang,
			SoTang:            item.SoTang,
			IDKhuVuc:          item.IDKhuVuc,
			TenKhuVuc:         item.TenKhuVuc,
			IDNguoiDung:       item.IDNguoiDung,
			TenNguoiDung:      item.TenNguoiDung,
			TrangThaiPhanCong: item.TrangThaiPhanCong,
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write assignment scope response: %v", err)
	}
}

func parseAssignmentScopeFilter(r *http.Request) (AssignmentScopeFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"id_nguoi_dung":        {},
		"so_tang":              {},
		"filter_so_tang":       {},
		"ten_khu_vuc":          {},
		"search":               {},
		"search_ten_khu_vuc":   {},
		"trang_thai_phan_cong": {},
		"filter_trang_thai_pc": {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return AssignmentScopeFilter{}, invalidAssignmentScopeQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return AssignmentScopeFilter{}, invalidAssignmentScopeQuery(key, "query parameter must be specified once")
		}
	}

	filter := AssignmentScopeFilter{}
	if value := strings.TrimSpace(query.Get("id_nguoi_dung")); value != "" {
		filter.UserID = &value
	}
	for _, key := range []string{"so_tang", "filter_so_tang"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			floorNumber, err := strconv.Atoi(value)
			if err != nil {
				return AssignmentScopeFilter{}, invalidAssignmentScopeQuery(key, "floor number must be an integer")
			}
			filter.FloorNumber = &floorNumber
			break
		}
	}
	for _, key := range []string{"ten_khu_vuc", "search", "search_ten_khu_vuc"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			filter.AreaName = &value
			break
		}
	}
	for _, key := range []string{"trang_thai_phan_cong", "filter_trang_thai_pc"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			filter.AssignmentStatus = &value
			break
		}
	}
	return filter, nil
}

func invalidAssignmentScopeQuery(field, message string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		message,
		map[string]any{"field": field},
	)
}
