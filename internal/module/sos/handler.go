package sos

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/authctx"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

type UseCase interface {
	ListCompleted(ctx context.Context, buildingID string, filter ListFilter) (*Page, error)
	ListActive(ctx context.Context, buildingID, userID string, filter ActiveFilter) (*ActivePage, error)
}

type Handler struct {
	service UseCase
}

type sosResponse struct {
	IDBaoCaoSOS       string     `json:"id_bao_cao_sos"`
	TenNguoiGui       string     `json:"ten_nguoi_gui"`
	SDTNguoiGui       string     `json:"sdt_nguoi_gui"`
	LoaiSOS           string     `json:"loai_sos"`
	MoTa              string     `json:"mo_ta"`
	HinhAnh           string     `json:"hinh_anh"`
	IDTang            *string    `json:"id_tang"`
	SoTang            *int       `json:"so_tang"`
	IDKhuVuc          *string    `json:"id_khu_vuc"`
	TenKhuVuc         string     `json:"ten_khu_vuc"`
	ThoiGianBatDau    time.Time  `json:"thoi_gian_bat_dau"`
	ThoiGianKetThuc   *time.Time `json:"thoi_gian_ket_thuc"`
	ThoiGianTiepNhan  *time.Time `json:"thoi_gian_tiep_nhan"`
	ThoiLuongXuLyGiay *int64     `json:"thoi_luong_xu_ly_giay"`
	TrangThai         string     `json:"trang_thai"`
	IDNguoiPhuTrach   *string    `json:"id_nguoi_phu_trach"`
	TenNguoiPhuTrach  string     `json:"ten_nguoi_phu_trach"`
	SDTNguoiPhuTrach  string     `json:"sdt_nguoi_phu_trach"`
	GhiChuXuLy        string     `json:"ghi_chu_xu_ly"`
}

type paginationResponse struct {
	Limit  int   `json:"limit"`
	Offset int64 `json:"offset"`
	Total  int   `json:"total"`
}

type listResponse struct {
	Items      []sosResponse      `json:"items"`
	Pagination paginationResponse `json:"pagination"`
	TotalItems int                `json:"total_items"`
}

type activeSOSResponse struct {
	IDBaoCaoSOS         string    `json:"id_bao_cao_sos"`
	TenNguoiGui         string    `json:"ten_nguoi_gui"`
	SDTNguoiGui         string    `json:"sdt_nguoi_gui"`
	LoaiSOS             string    `json:"loai_sos"`
	TrangThai           string    `json:"trang_thai"`
	MoTa                string    `json:"mo_ta"`
	HinhAnh             string    `json:"hinh_anh"`
	ThoiGianGui         time.Time `json:"thoi_gian_gui"`
	IDTang              *string   `json:"id_tang"`
	SoTang              *int      `json:"so_tang"`
	IDKhuVuc            *string   `json:"id_khu_vuc"`
	TenKhuVuc           string    `json:"ten_khu_vuc"`
	ThuocPhanCongCuaToi bool      `json:"thuoc_phan_cong_cua_toi"`
}

type activeListResponse struct {
	Items      []activeSOSResponse `json:"items"`
	Pagination paginationResponse  `json:"pagination"`
	TotalItems int                 `json:"total_items"`
}

func NewHandler(service UseCase) (*Handler, error) {
	if service == nil {
		return nil, errors.New("SOS service is required")
	}
	return &Handler{service: service}, nil
}

func (h *Handler) ListCompleted(w http.ResponseWriter, r *http.Request) {
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

	filter, err := parseListFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	page, err := h.service.ListCompleted(r.Context(), principal.IDToaNha, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if page == nil {
		httpx.WriteError(w, errors.New("SOS service returned nil page"))
		return
	}

	items := make([]sosResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = sosResponse{
			IDBaoCaoSOS: item.IDBaoCaoSOS, TenNguoiGui: item.TenNguoiGui,
			SDTNguoiGui: item.SDTNguoiGui, LoaiSOS: item.LoaiSOS,
			MoTa: item.MoTa, HinhAnh: item.HinhAnh, IDTang: item.IDTang,
			SoTang: item.SoTang, IDKhuVuc: item.IDKhuVuc, TenKhuVuc: item.TenKhuVuc,
			ThoiGianBatDau: item.ThoiGianBatDau, ThoiGianKetThuc: item.ThoiGianKetThuc,
			ThoiGianTiepNhan: item.ThoiGianTiepNhan, ThoiLuongXuLyGiay: item.ThoiLuongXuLyGiay,
			TrangThai: item.TrangThai, IDNguoiPhuTrach: item.IDNguoiPhuTrach,
			TenNguoiPhuTrach: item.TenNguoiPhuTrach, SDTNguoiPhuTrach: item.SDTNguoiPhuTrach,
			GhiChuXuLy: item.GhiChuXuLy,
		}
	}
	response := listResponse{
		Items:      items,
		Pagination: paginationResponse{Limit: page.Limit, Offset: page.Offset, Total: page.Total},
		TotalItems: page.Total,
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write SOS list response: %v", err)
	}
}

func (h *Handler) ListActive(w http.ResponseWriter, r *http.Request) {
	principal, ok := authctx.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.IDToaNha) == "" || strings.TrimSpace(principal.UserID) == "" {
		httpx.WriteError(w, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building and user scope are missing from authentication token",
			nil,
		))
		return
	}

	filter, err := parseActiveFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	page, err := h.service.ListActive(r.Context(), principal.IDToaNha, principal.UserID, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if page == nil {
		httpx.WriteError(w, errors.New("SOS service returned nil active page"))
		return
	}

	items := make([]activeSOSResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = activeSOSResponse{
			IDBaoCaoSOS:         item.IDBaoCaoSOS,
			TenNguoiGui:         item.TenNguoiGui,
			SDTNguoiGui:         item.SDTNguoiGui,
			LoaiSOS:             item.LoaiSOS,
			TrangThai:           item.TrangThai,
			MoTa:                item.MoTa,
			HinhAnh:             item.HinhAnh,
			ThoiGianGui:         item.ThoiGianGui,
			IDTang:              item.IDTang,
			SoTang:              item.SoTang,
			IDKhuVuc:            item.IDKhuVuc,
			TenKhuVuc:           item.TenKhuVuc,
			ThuocPhanCongCuaToi: item.ThuocPhanCongCuaToi,
		}
	}
	response := activeListResponse{
		Items: items,
		Pagination: paginationResponse{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
		TotalItems: page.Total,
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write active SOS list response: %v", err)
	}
}

func parseActiveFilter(r *http.Request) (ActiveFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"loai_sos":            {},
		"trang_thai":          {},
		"chi_thuoc_phan_cong": {},
		"limit":               {},
		"offset":              {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return ActiveFilter{}, invalidQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return ActiveFilter{}, invalidQuery(key, "query parameter must be specified once")
		}
	}

	filter := ActiveFilter{Limit: DefaultPageSize}
	if value := strings.TrimSpace(query.Get("loai_sos")); value != "" {
		filter.LoaiSOS = &value
	}
	if value := strings.TrimSpace(query.Get("trang_thai")); value != "" {
		filter.TrangThai = &value
	}
	if value := strings.TrimSpace(query.Get("chi_thuoc_phan_cong")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return ActiveFilter{}, invalidQuery("chi_thuoc_phan_cong", "value must be true or false")
		}
		filter.ChiThuocPhanCong = parsed
	}
	if value := strings.TrimSpace(query.Get("limit")); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number < 1 || number > MaxPageSize {
			return ActiveFilter{}, invalidQuery("limit", "limit must be an integer between 1 and 100")
		}
		filter.Limit = number
	}
	if value := strings.TrimSpace(query.Get("offset")); value != "" {
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil || number < 0 {
			return ActiveFilter{}, invalidQuery("offset", "offset must be a non-negative integer")
		}
		filter.Offset = number
	}
	return filter, nil
}

func parseListFilter(r *http.Request) (ListFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"loai_sos": {}, "from_time": {}, "to_time": {},
		"keyword": {}, "limit": {}, "offset": {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return ListFilter{}, invalidQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return ListFilter{}, invalidQuery(key, "query parameter must be specified once")
		}
	}

	filter := ListFilter{Limit: DefaultPageSize}
	if value := strings.TrimSpace(query.Get("loai_sos")); value != "" {
		filter.LoaiSOS = &value
	}
	if value := strings.TrimSpace(query.Get("keyword")); value != "" {
		filter.Keyword = &value
	}
	for key, target := range map[string]**time.Time{"from_time": &filter.FromTime, "to_time": &filter.ToTime} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			parsed, err := parseTimestamp(value)
			if err != nil {
				return ListFilter{}, invalidQuery(key, "timestamp must use RFC3339 format")
			}
			*target = &parsed
		}
	}
	if value := strings.TrimSpace(query.Get("limit")); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number < 1 || number > MaxPageSize {
			return ListFilter{}, invalidQuery("limit", "limit must be an integer between 1 and 100")
		}
		filter.Limit = number
	}
	if value := strings.TrimSpace(query.Get("offset")); value != "" {
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil || number < 0 {
			return ListFilter{}, invalidQuery("offset", "offset must be a non-negative integer")
		}
		filter.Offset = number
	}
	return filter, nil
}

func parseTimestamp(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
}

func invalidQuery(field, message string) error {
	return apperror.New(http.StatusBadRequest, apperror.AppErrorInvalidArgument, message, map[string]any{"field": field})
}
