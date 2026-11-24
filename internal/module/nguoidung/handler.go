package nguoidung

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

const maxBatchRequestBytes = 1 << 20

type BatchUserUseCase interface {
	CreateBatch(
		ctx context.Context,
		actorID string,
		actorRoleCode string,
		buildingID string,
		requests []BatchUserRequest,
	) ([]CreatedUser, error)
}

type NguoiDungHandler struct {
	batchUsers BatchUserUseCase
	residents  ResidentUseCase
}

type ResidentUseCase interface {
	ListResidents(ctx context.Context, buildingID string, filter ResidentFilter) (*ResidentPage, error)
	ListSupervisorActors(ctx context.Context, buildingID string, filter SupervisorActorFilter) (*SupervisorActorPage, error)
}

type batchCreateRequest struct {
	IDToaNha string              `json:"id_toa_nha"`
	Users    []createUserRequest `json:"users"`
}

type createUserRequest struct {
	TenNguoiDung string                 `json:"ten_nguoi_dung"`
	Email        string                 `json:"email"`
	SoDienThoai  string                 `json:"so_dien_thoai"`
	MatKhau      string                 `json:"mat_khau"`
	IDVaiTro     string                 `json:"id_vai_tro"`
	KyHieuVaiTro string                 `json:"ky_hieu_vai_tro"`
	IDKhuVuc     string                 `json:"id_khu_vuc"`
	PhanCong     *managerAssignmentJSON `json:"phan_cong"`
}

type managerAssignmentJSON struct {
	IDTang   string `json:"id_tang"`
	IDKhuVuc string `json:"id_khu_vuc"`
}

type createdUserResponse struct {
	IDNguoiDung  string `json:"id_nguoi_dung"`
	IDVaiTro     string `json:"id_vai_tro"`
	KyHieuVaiTro string `json:"ky_hieu_vai_tro"`
	TenNguoiDung string `json:"ten_nguoi_dung"`
	Email        string `json:"email"`
	SoDienThoai  string `json:"so_dien_thoai,omitempty"`
	TrangThai    string `json:"trang_thai"`
}

func NewNguoiDungHandler(
	batchUsers BatchUserUseCase,
	residentServices ...ResidentUseCase,
) (*NguoiDungHandler, error) {
	if batchUsers == nil {
		return nil, errors.New("batch user service is required")
	}
	handler := &NguoiDungHandler{batchUsers: batchUsers}
	if len(residentServices) > 0 {
		handler.residents = residentServices[0]
	}
	return handler, nil
}

func (h *NguoiDungHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	principal, ok := authctx.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, apperror.New(
			http.StatusUnauthorized,
			apperror.AppErrorUnauthenticated,
			"authentication is required",
			nil,
		))
		return
	}

	if r.URL.RawQuery != "" {
		httpx.WriteError(w, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"query parameters are not supported",
			nil,
		))
		return
	}

	var request batchCreateRequest
	if err := httpx.DecodeJSON(w, r, maxBatchRequestBytes, &request); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if len(request.Users) == 0 || len(request.Users) > MaxBatchUsers {
		httpx.WriteError(w, ErrBatchSizeInvalid)
		return
	}

	users := make([]BatchUserRequest, len(request.Users))
	for i, user := range request.Users {
		users[i] = BatchUserRequest{
			TenNguoiDung: strings.TrimSpace(user.TenNguoiDung),
			Email:        user.Email,
			SoDienThoai:  user.SoDienThoai,
			MatKhau:      user.MatKhau,
			IDVaiTro:     user.IDVaiTro,
			KyHieuVaiTro: user.KyHieuVaiTro,
			IDKhuVuc:     user.IDKhuVuc,
		}
		if user.PhanCong != nil {
			users[i].PhanCong = &ManagerAssignment{
				IDTang:   user.PhanCong.IDTang,
				IDKhuVuc: user.PhanCong.IDKhuVuc,
			}
		}
	}

	created, err := h.batchUsers.CreateBatch(
		r.Context(),
		principal.UserID,
		strings.ToUpper(strings.TrimSpace(principal.RoleID)),
		request.IDToaNha,
		users,
	)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]createdUserResponse, len(created))
	for i, user := range created {
		response[i] = createdUserResponse{
			IDNguoiDung:  user.IDNguoiDung,
			IDVaiTro:     user.IDVaiTro,
			KyHieuVaiTro: user.KyHieuVaiTro,
			TenNguoiDung: user.TenNguoiDung,
			Email:        user.Email,
			SoDienThoai:  user.SoDienThoai,
			TrangThai:    string(user.TrangThai),
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusCreated, response); err != nil {
		log.Printf("write batch user creation response: %v", err)
	}
}

type residentResponse struct {
	IDNguoiDung  string `json:"id_nguoi_dung"`
	TenNguoiDung string `json:"ten_nguoi_dung"`
	Email        string `json:"email"`
	SoDienThoai  string `json:"so_dien_thoai"`
	TrangThai    string `json:"trang_thai"`
	NgayTao      string `json:"ngay_tao"`
	IDKhuVuc     string `json:"id_khu_vuc"`
	TenKhuVuc    string `json:"ten_khu_vuc"`
	IDTang       string `json:"id_tang"`
	SoTang       int    `json:"so_tang"`
}

type residentPaginationResponse struct {
	Limit  int   `json:"limit"`
	Offset int64 `json:"offset"`
	Total  int   `json:"total"`
}

type residentListResponse struct {
	Items      []residentResponse         `json:"items"`
	Pagination residentPaginationResponse `json:"pagination"`
	TotalItems int                        `json:"total_items"`
	Floors     []residentFloorResponse    `json:"floors,omitempty"`
}

type residentFloorResponse struct {
	IDTang string `json:"id_tang"`
	SoTang int    `json:"so_tang"`
}

type supervisorActorResponse struct {
	IDNguoiDung       string `json:"id_nguoi_dung"`
	TenNguoiDung      string `json:"ten_nguoi_dung"`
	Email             string `json:"email"`
	SoDienThoai       string `json:"so_dien_thoai"`
	TrangThai         string `json:"trang_thai"`
	NgayTao           string `json:"ngay_tao"`
	IDVaiTro          string `json:"id_vai_tro"`
	KyHieuVaiTro      string `json:"ky_hieu_vai_tro"`
	TenVaiTro         string `json:"ten_vai_tro"`
	TrangThaiPhanCong string `json:"trang_thai_phan_cong"`
}

type supervisorActorListResponse struct {
	Items      []supervisorActorResponse  `json:"items"`
	Pagination residentPaginationResponse `json:"pagination"`
	TotalItems int                        `json:"total_items"`
}

func (h *NguoiDungHandler) ListSupervisorActors(w http.ResponseWriter, r *http.Request) {
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
	if h.residents == nil {
		httpx.WriteError(w, errors.New("supervisor actor service is not configured"))
		return
	}

	filter, err := parseSupervisorActorFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	page, err := h.residents.ListSupervisorActors(r.Context(), principal.IDToaNha, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if page == nil {
		httpx.WriteError(w, errors.New("supervisor actor service returned nil page"))
		return
	}

	items := make([]supervisorActorResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = supervisorActorResponse{
			IDNguoiDung:       item.IDNguoiDung,
			TenNguoiDung:      item.TenNguoiDung,
			Email:             item.Email,
			SoDienThoai:       item.SoDienThoai,
			TrangThai:         item.TrangThai,
			NgayTao:           item.NgayTao,
			IDVaiTro:          item.IDVaiTro,
			KyHieuVaiTro:      item.KyHieuVaiTro,
			TenVaiTro:         item.TenVaiTro,
			TrangThaiPhanCong: item.TrangThaiPhanCong,
		}
	}
	response := supervisorActorListResponse{
		Items: items,
		Pagination: residentPaginationResponse{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
		TotalItems: page.Total,
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write supervisor actor response: %v", err)
	}
}

func (h *NguoiDungHandler) ListResidents(w http.ResponseWriter, r *http.Request) {
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
	if h.residents == nil {
		httpx.WriteError(w, errors.New("resident service is not configured"))
		return
	}

	filter, err := parseResidentFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	filter.IncludeFloors = hasResidentPaginationOnlyQuery(r) && filter.Offset == 0
	page, err := h.residents.ListResidents(r.Context(), principal.IDToaNha, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if page == nil {
		httpx.WriteError(w, errors.New("resident service returned nil page"))
		return
	}

	items := make([]residentResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = residentResponse{
			IDNguoiDung:  item.IDNguoiDung,
			TenNguoiDung: item.TenNguoiDung,
			Email:        item.Email,
			SoDienThoai:  item.SoDienThoai,
			TrangThai:    item.TrangThai,
			NgayTao:      item.NgayTao,
			IDKhuVuc:     item.IDKhuVuc,
			TenKhuVuc:    item.TenKhuVuc,
			IDTang:       item.IDTang,
			SoTang:       item.SoTang,
		}
	}
	response := residentListResponse{
		Items: items,
		Pagination: residentPaginationResponse{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
		TotalItems: page.Total,
	}
	if filter.IncludeFloors {
		response.Floors = make([]residentFloorResponse, len(page.Floors))
		for i, floor := range page.Floors {
			response.Floors[i] = residentFloorResponse{
				IDTang: floor.IDTang,
				SoTang: floor.SoTang,
			}
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write building resident response: %v", err)
	}
}

func parseSupervisorActorFilter(r *http.Request) (SupervisorActorFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"search":               {},
		"search_keyword":       {},
		"trang_thai_phan_cong": {},
		"trang_thai":           {},
		"limit":                {},
		"offset":               {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return SupervisorActorFilter{}, invalidResidentQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return SupervisorActorFilter{}, invalidResidentQuery(key, "query parameter must be specified once")
		}
	}

	filter := SupervisorActorFilter{Limit: DefaultSupervisorActorPageSize}
	if values, ok := query["limit"]; ok {
		value, err := strconv.Atoi(values[0])
		if err != nil || value < 1 || value > MaxSupervisorActorPageSize {
			return SupervisorActorFilter{}, invalidResidentQuery("limit", "limit must be an integer between 1 and 100")
		}
		filter.Limit = value
	}
	if values, ok := query["offset"]; ok {
		value, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil || value < 0 {
			return SupervisorActorFilter{}, invalidResidentQuery("offset", "offset must be a non-negative integer")
		}
		filter.Offset = value
	}
	for _, key := range []string{"search", "search_keyword"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			filter.SearchKeyword = &value
			break
		}
	}
	for _, key := range []string{"trang_thai_phan_cong", "trang_thai"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			filter.Status = &value
			break
		}
	}
	return filter, nil
}

func parseResidentFilter(r *http.Request) (ResidentFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"id_tang":    {},
		"id_khu_vuc": {},
		"trang_thai": {},
		"search":     {},
		"limit":      {},
		"offset":     {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return ResidentFilter{}, invalidResidentQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return ResidentFilter{}, invalidResidentQuery(key, "query parameter must be specified once")
		}
	}

	filter := ResidentFilter{Limit: DefaultResidentPageSize}
	if values, ok := query["limit"]; ok {
		value, err := strconv.Atoi(values[0])
		if err != nil || value < 1 || value > MaxResidentPageSize {
			return ResidentFilter{}, invalidResidentQuery("limit", "limit must be an integer between 1 and 100")
		}
		filter.Limit = value
	}
	if values, ok := query["offset"]; ok {
		value, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil || value < 0 {
			return ResidentFilter{}, invalidResidentQuery("offset", "offset must be a non-negative integer")
		}
		filter.Offset = value
	}
	if value := strings.TrimSpace(query.Get("id_tang")); value != "" {
		filter.FloorID = &value
	}
	if value := strings.TrimSpace(query.Get("id_khu_vuc")); value != "" {
		filter.AreaID = &value
	}
	if value := strings.TrimSpace(query.Get("trang_thai")); value != "" {
		filter.Status = &value
	}
	if value := strings.TrimSpace(query.Get("search")); value != "" {
		filter.SearchKeyword = &value
	}
	return filter, nil
}

func hasResidentPaginationOnlyQuery(r *http.Request) bool {
	if r.URL.RawQuery == "" {
		return true
	}
	for key := range r.URL.Query() {
		if key != "limit" && key != "offset" {
			return false
		}
	}
	return true
}

func invalidResidentQuery(field, message string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		message,
		map[string]any{"field": field},
	)
}
