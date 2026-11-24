package canhbao

import (
	"context"
	"encoding/json"
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
	List(ctx context.Context, buildingID string, filter ListFilter) (*Page, error)
	ListActive(ctx context.Context, buildingID, userID string, filter ActiveFilter) (*ActivePage, error)
}

type Handler struct {
	service UseCase
}

type alertResponse struct {
	IDCanhBao        string     `json:"id_canh_bao"`
	MucDo            string     `json:"muc_do"`
	TrangThaiXuLy    string     `json:"trang_thai_xu_ly"`
	MoTa             string     `json:"mo_ta"`
	ThoiGianPhatHien time.Time  `json:"thoi_gian_phat_hien"`
	ThoiGianXacNhan  *time.Time `json:"thoi_gian_xac_nhan"`
	ThoiGianTiepNhan *time.Time `json:"thoi_gian_tiep_nhan"`
	ThoiGianKetThuc  *time.Time `json:"thoi_gian_ket_thuc"`
	IDTang           string     `json:"id_tang"`
	SoTang           int        `json:"so_tang"`
	IDKhuVuc         string     `json:"id_khu_vuc"`
	TenKhuVuc        string     `json:"ten_khu_vuc"`
}

type alertPaginationResponse struct {
	Limit  int   `json:"limit"`
	Offset int64 `json:"offset"`
	Total  int   `json:"total"`
}

type alertListResponse struct {
	Items      []alertResponse         `json:"items"`
	Pagination alertPaginationResponse `json:"pagination"`
	TotalItems int                     `json:"total_items"`
}

type activeAlertResponse struct {
	IDCanhBao               string          `json:"id_canh_bao"`
	MucDo                   string          `json:"muc_do"`
	TrangThai               string          `json:"trang_thai"`
	MoTa                    string          `json:"mo_ta"`
	ThoiGianPhatHien        time.Time       `json:"thoi_gian_phat_hien"`
	ThoiGianXacNhan         *time.Time      `json:"thoi_gian_xac_nhan"`
	ThoiGianTiepNhan        *time.Time      `json:"thoi_gian_tiep_nhan"`
	IDTang                  string          `json:"id_tang"`
	SoTang                  int             `json:"so_tang"`
	IDKhuVuc                string          `json:"id_khu_vuc"`
	TenKhuVuc               string          `json:"ten_khu_vuc"`
	ThuocPhanCongCuaToi     bool            `json:"thuoc_phan_cong_cua_toi"`
	IDNguoiTiepNhan         *string         `json:"id_nguoi_tiep_nhan"`
	TenNguoiTiepNhan        string          `json:"ten_nguoi_tiep_nhan"`
	SDTNguoiTiepNhan        string          `json:"sdt_nguoi_tiep_nhan"`
	ThoiGianThaoTacTiepNhan *time.Time      `json:"thoi_gian_thao_tac_tiep_nhan"`
	DanhSachSuKien          json.RawMessage `json:"danh_sach_su_kien"`
}

type activeAlertListResponse struct {
	Items      []activeAlertResponse   `json:"items"`
	Pagination alertPaginationResponse `json:"pagination"`
	TotalItems int                     `json:"total_items"`
}

func NewHandler(service UseCase) (*Handler, error) {
	if service == nil {
		return nil, errors.New("alert service is required")
	}
	return &Handler{service: service}, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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
	page, err := h.service.List(r.Context(), principal.IDToaNha, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if page == nil {
		httpx.WriteError(w, errors.New("alert service returned nil page"))
		return
	}

	items := make([]alertResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = alertResponse{
			IDCanhBao: item.IDCanhBao, MucDo: item.MucDo,
			TrangThaiXuLy: item.TrangThai, MoTa: item.MoTa,
			ThoiGianPhatHien: item.ThoiGianPhatHien,
			ThoiGianXacNhan:  item.ThoiGianXacNhan,
			ThoiGianTiepNhan: item.ThoiGianTiepNhan,
			ThoiGianKetThuc:  item.ThoiGianKetThuc,
			IDTang:           item.IDTang, SoTang: item.SoTang,
			IDKhuVuc: item.IDKhuVuc, TenKhuVuc: item.TenKhuVuc,
		}
	}
	response := alertListResponse{
		Items: items,
		Pagination: alertPaginationResponse{
			Limit: page.Limit, Offset: page.Offset, Total: page.Total,
		},
		TotalItems: page.Total,
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write alert list response: %v", err)
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
		httpx.WriteError(w, errors.New("alert service returned nil active page"))
		return
	}

	items := make([]activeAlertResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = activeAlertResponse{
			IDCanhBao:               item.IDCanhBao,
			MucDo:                   item.MucDo,
			TrangThai:               item.TrangThai,
			MoTa:                    item.MoTa,
			ThoiGianPhatHien:        item.ThoiGianPhatHien,
			ThoiGianXacNhan:         item.ThoiGianXacNhan,
			ThoiGianTiepNhan:        item.ThoiGianTiepNhan,
			IDTang:                  item.IDTang,
			SoTang:                  item.SoTang,
			IDKhuVuc:                item.IDKhuVuc,
			TenKhuVuc:               item.TenKhuVuc,
			ThuocPhanCongCuaToi:     item.ThuocPhanCongCuaToi,
			IDNguoiTiepNhan:         item.IDNguoiTiepNhan,
			TenNguoiTiepNhan:        item.TenNguoiTiepNhan,
			SDTNguoiTiepNhan:        item.SDTNguoiTiepNhan,
			ThoiGianThaoTacTiepNhan: item.ThoiGianThaoTacTiepNhan,
			DanhSachSuKien:          item.DanhSachSuKien,
		}
	}
	response := activeAlertListResponse{
		Items: items,
		Pagination: alertPaginationResponse{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
		TotalItems: page.Total,
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write active alert list response: %v", err)
	}
}

func parseActiveFilter(r *http.Request) (ActiveFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"trang_thai":          {},
		"chi_thuoc_phan_cong": {},
		"limit":               {},
		"offset":              {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return ActiveFilter{}, invalidAlertQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return ActiveFilter{}, invalidAlertQuery(key, "query parameter must be specified once")
		}
	}

	filter := ActiveFilter{Limit: DefaultPageSize}
	if value := strings.TrimSpace(query.Get("trang_thai")); value != "" {
		filter.TrangThai = &value
	}
	if value := strings.TrimSpace(query.Get("chi_thuoc_phan_cong")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return ActiveFilter{}, invalidAlertQuery("chi_thuoc_phan_cong", "value must be true or false")
		}
		filter.ChiThuocPhanCong = parsed
	}
	if value := strings.TrimSpace(query.Get("limit")); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number < 1 || number > MaxPageSize {
			return ActiveFilter{}, invalidAlertQuery("limit", "limit must be an integer between 1 and 100")
		}
		filter.Limit = number
	}
	if value := strings.TrimSpace(query.Get("offset")); value != "" {
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil || number < 0 {
			return ActiveFilter{}, invalidAlertQuery("offset", "offset must be a non-negative integer")
		}
		filter.Offset = number
	}
	return filter, nil
}

func parseListFilter(r *http.Request) (ListFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"muc_do": {}, "trang_thai": {},
		"id_tang": {}, "id_khu_vuc": {},
		"from_thoi_gian_phat_hien": {}, "to_thoi_gian_phat_hien": {},
		"limit": {}, "offset": {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return ListFilter{}, invalidAlertQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return ListFilter{}, invalidAlertQuery(key, "query parameter must be specified once")
		}
	}

	filter := ListFilter{Limit: DefaultPageSize}
	if value := strings.TrimSpace(query.Get("muc_do")); value != "" {
		filter.MucDo = &value
	}
	if value := strings.TrimSpace(query.Get("trang_thai")); value != "" {
		filter.TrangThai = &value
	}
	if value := strings.TrimSpace(query.Get("id_khu_vuc")); value != "" {
		filter.IDKhuVuc = &value
	}
	if value := strings.TrimSpace(query.Get("id_tang")); value != "" {
		filter.IDTang = &value
	}
	if value := strings.TrimSpace(query.Get("from_thoi_gian_phat_hien")); value != "" {
		parsed, err := parseTimestamp(value)
		if err != nil {
			return ListFilter{}, invalidAlertQuery("from_thoi_gian_phat_hien", "timestamp must use RFC3339 format")
		}
		filter.FromPhatHien = &parsed
	}
	if value := strings.TrimSpace(query.Get("to_thoi_gian_phat_hien")); value != "" {
		parsed, err := parseTimestamp(value)
		if err != nil {
			return ListFilter{}, invalidAlertQuery("to_thoi_gian_phat_hien", "timestamp must use RFC3339 format")
		}
		filter.ToPhatHien = &parsed
	}
	if value := strings.TrimSpace(query.Get("limit")); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil || number < 1 || number > MaxPageSize {
			return ListFilter{}, invalidAlertQuery("limit", "limit must be an integer between 1 and 100")
		}
		filter.Limit = number
	}
	if value := strings.TrimSpace(query.Get("offset")); value != "" {
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil || number < 0 {
			return ListFilter{}, invalidAlertQuery("offset", "offset must be a non-negative integer")
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

func invalidAlertQuery(field, message string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		message,
		map[string]any{"field": field},
	)
}
