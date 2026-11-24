package toanha

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

type ToaNhaUseCase interface {
	FindByID(ctx context.Context, id string) (*ToaNha, error)
	List(ctx context.Context, filter ListFilter) (*Page, error)
}

type ToaNhaHandler struct {
	service ToaNhaUseCase
}

type toaNhaResponse struct {
	IDToaNha     string `json:"id_toa_nha"`
	IDNguoiChu   string `json:"id_nguoi_chu"`
	IDLoaiMoHinh string `json:"id_loai_mo_hinh"`
	SoTang       int    `json:"so_tang"`
	TenToaNha    string `json:"ten_toa_nha"`
	NgayTao      string `json:"ngay_tao"`
	DiaChi       string `json:"dia_chi"`
	NgayCapNhat  string `json:"ngay_cap_nhat"`
	MoTa         string `json:"mo_ta"`
	TrangThai    string `json:"trang_thai"`
}

type listToaNhaResponse struct {
	Items      []toaNhaResponse   `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

type paginationResponse struct {
	Limit  int   `json:"limit"`
	Offset int64 `json:"offset"`
	Total  int64 `json:"total"`
}

func NewToaNhaHandler(service ToaNhaUseCase) (*ToaNhaHandler, error) {
	if service == nil {
		return nil, errors.New("toa nha service is required")
	}

	return &ToaNhaHandler{
		service: service,
	}, nil
}

func (h *ToaNhaHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseListFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	page, err := h.service.List(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if page == nil {
		httpx.WriteError(w, errors.New("toa nha service returned nil page"))
		return
	}

	items := make([]toaNhaResponse, len(page.Items))
	for i := range page.Items {
		items[i] = toToaNhaResponse(&page.Items[i])
	}

	response := listToaNhaResponse{
		Items: items,
		Pagination: paginationResponse{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write list toa nha response: %v", err)
	}
}

func (h *ToaNhaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteError(w, errors.New("toa nha ID is required"))
		return
	}

	toaNha, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if toaNha == nil {
		httpx.WriteError(w, errors.New("toa nha service returned nil result"))
		return
	}

	response := toToaNhaResponse(toaNha)

	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write get toa nha response: %v", err)
	}
}

func toToaNhaResponse(toaNha *ToaNha) toaNhaResponse {
	return toaNhaResponse{
		IDToaNha:     toaNha.IDToaNha,
		IDNguoiChu:   toaNha.IDNguoiChu,
		IDLoaiMoHinh: toaNha.IDLoaiMoHinh,
		SoTang:       toaNha.SoTang,
		TenToaNha:    toaNha.TenToaNha,
		NgayTao:      toaNha.NgayTao.Format("2006-01-02T15:04:05.999999999Z07:00"),
		DiaChi:       toaNha.DiaChi,
		NgayCapNhat:  toaNha.NgayCapNhat.Format("2006-01-02T15:04:05.999999999Z07:00"),
		MoTa:         toaNha.MoTa,
		TrangThai:    toaNha.TrangThai,
	}
}

func parseListFilter(r *http.Request) (ListFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"limit":           {},
		"offset":          {},
		"trang_thai":      {},
		"id_loai_mo_hinh": {},
		"ten_toa_nha":     {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return ListFilter{}, invalidQueryParameter(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return ListFilter{}, invalidQueryParameter(key, "query parameter must be specified once")
		}
	}

	filter := ListFilter{
		Limit:        DefaultPageSize,
		TrangThai:    strings.TrimSpace(query.Get("trang_thai")),
		IDLoaiMoHinh: strings.TrimSpace(query.Get("id_loai_mo_hinh")),
		TenToaNha:    strings.TrimSpace(query.Get("ten_toa_nha")),
	}
	if value, ok := query["limit"]; ok {
		limit, err := strconv.Atoi(value[0])
		if err != nil || limit < 1 || limit > MaxPageSize {
			return ListFilter{}, invalidQueryParameter(
				"limit",
				"limit must be an integer between 1 and 100",
			)
		}
		filter.Limit = limit
	}
	if value, ok := query["offset"]; ok {
		offset, err := strconv.ParseInt(value[0], 10, 64)
		if err != nil || offset < 0 {
			return ListFilter{}, invalidQueryParameter("offset", "offset must be a non-negative integer")
		}
		filter.Offset = offset
	}
	return filter, nil
}

func invalidQueryParameter(field, message string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		message,
		map[string]any{"field": field},
	)
}
