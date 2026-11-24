package khuvuc

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

type UseCase interface {
	ListByFloorID(ctx context.Context, floorID string) ([]KhuVuc, error)
	ListShortByFloorID(ctx context.Context, floorID string) ([]KhuVucShort, error)
	ListMonitoring(ctx context.Context, buildingID string, filter MonitoringFilter) ([]RoomMonitoring, error)
	ListFloorMonitoring(ctx context.Context, buildingID string) ([]FloorMonitoring, error)
	ListBuildingAreas(ctx context.Context, buildingID string, filter BuildingAreaFilter) (*BuildingAreaPage, error)
}

type Handler struct {
	service UseCase
}

type khuVucResponse struct {
	IDKhuVuc     string `json:"id_khu_vuc"`
	IDTang       string `json:"id_tang"`
	IDLoaiKhuVuc string `json:"id_loai_khu_vuc"`
	TenKhuVuc    string `json:"ten_khu_vuc"`
	MoTa         string `json:"mo_ta,omitempty"`
}

type khuVucShortResponse struct {
	IDKhuVuc      string `json:"id_khu_vuc"`
	IDLoaiKhuVuc  string `json:"id_loai_khu_vuc"`
	TenLoaiKhuVuc string `json:"ten_loai_khu_vuc"`
	TenKhuVuc     string `json:"ten_khu_vuc"`
}

type roomMonitoringResponse struct {
	FloorNumber       int      `json:"so_tang"`
	FloorID           string   `json:"id_tang"`
	AreaID            string   `json:"id_khu_vuc"`
	AreaName          string   `json:"ten_khu_vuc"`
	AreaDescription   string   `json:"mo_ta_khu_vuc"`
	AreaTypeName      string   `json:"ten_loai_khu_vuc"`
	AlertLevel        string   `json:"muc_do_canh_bao,omitempty"`
	AlertStatus       string   `json:"trang_thai_canh_bao,omitempty"`
	AlertDescription  string   `json:"mo_ta_canh_bao,omitempty"`
	HasFireAlert      bool     `json:"co_canh_bao"`
	HasHardwareError  bool     `json:"co_loi_phan_cung"`
	HardwareStatus    string   `json:"trang_thai_phan_cung"`
	Temperature       *float64 `json:"nhiet_do"`
	TemperatureUnit   string   `json:"don_vi_nhiet_do"`
	SmokeDensity      *float64 `json:"nong_do_khoi"`
	SmokeUnit         string   `json:"don_vi_khoi"`
	StatusDescription string   `json:"mo_ta_tinh_trang"`
}

type floorMonitoringResponse struct {
	FloorID     string `json:"id_tang"`
	FloorNumber int    `json:"so_tang"`
	AreaCount   int    `json:"so_luong_khu_vuc"`
	StatusColor string `json:"mau_trang_thai"`
	StatusCode  string `json:"ma_trang_thai"`
}

type buildingAreaResponse struct {
	AreaID          string `json:"id_khu_vuc"`
	AreaName        string `json:"ten_khu_vuc"`
	AreaDescription string `json:"mo_ta_khu_vuc"`
	FloorID         string `json:"id_tang"`
	FloorNumber     int    `json:"so_tang"`
	AreaTypeID      string `json:"id_loai_khu_vuc"`
	AreaTypeName    string `json:"ten_loai_khu_vuc"`
	ResidentCount   int    `json:"so_luong_cu_dan"`
	StatusColor     string `json:"mau_trang_thai"`
	StatusCode      string `json:"ma_trang_thai"`
}

type buildingFloorResponse struct {
	FloorID     string `json:"id_tang"`
	FloorNumber int    `json:"so_tang"`
}

type buildingAreaTypeResponse struct {
	AreaTypeID   string `json:"id_loai_khu_vuc"`
	AreaTypeName string `json:"ten_loai_khu_vuc"`
}

type buildingAreaListResponse struct {
	Items      []buildingAreaResponse     `json:"items"`
	TotalCount int                        `json:"total_count"`
	Floors     []buildingFloorResponse    `json:"floors"`
	AreaTypes  []buildingAreaTypeResponse `json:"area_types"`
	Pagination buildingAreaPagination     `json:"pagination"`
}

type buildingAreaPagination struct {
	Limit  int   `json:"limit"`
	Offset int64 `json:"offset"`
	Total  int   `json:"total"`
}

func NewHandler(service UseCase) (*Handler, error) {
	if service == nil {
		return nil, errors.New("area service is required")
	}
	return &Handler{service: service}, nil
}

func (h *Handler) ListByFloorID(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		httpx.WriteError(w, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"query parameters are not supported",
			nil,
		))
		return
	}

	items, err := h.service.ListByFloorID(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]khuVucResponse, len(items))
	for i, item := range items {
		response[i] = khuVucResponse{
			IDKhuVuc:     item.IDKhuVuc,
			IDTang:       item.IDTang,
			IDLoaiKhuVuc: item.IDLoaiKhuVuc,
			TenKhuVuc:    item.TenKhuVuc,
			MoTa:         item.MoTa,
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write list khu vuc response: %v", err)
	}
}

func (h *Handler) ListShortDataByFloorID(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		httpx.WriteError(w, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"query parameters are not supported",
			nil,
		))
		return
	}

	items, err := h.service.ListShortByFloorID(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]khuVucShortResponse, len(items))
	for i, item := range items {
		response[i] = khuVucShortResponse{
			IDKhuVuc:      item.IDKhuVuc,
			TenKhuVuc:     item.TenKhuVuc,
			IDLoaiKhuVuc:  item.IDLoaiKhuVuc,
			TenLoaiKhuVuc: item.TenLoaiKhuVuc,
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write list khu vuc response: %v", err)
	}
}

func (h *Handler) ListMonitoring(w http.ResponseWriter, r *http.Request) {
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

	filter, err := parseMonitoringFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	items, err := h.service.ListMonitoring(r.Context(), principal.IDToaNha, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]roomMonitoringResponse, len(items))
	for i, item := range items {
		response[i] = toRoomMonitoringResponse(item)
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write area monitoring response: %v", err)
	}
}

func (h *Handler) ListFloorMonitoring(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	items, err := h.service.ListFloorMonitoring(r.Context(), principal.IDToaNha)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]floorMonitoringResponse, len(items))
	for i, item := range items {
		response[i] = floorMonitoringResponse{
			FloorID:     item.IDTang,
			FloorNumber: item.SoTang,
			AreaCount:   item.SoLuongKhuVuc,
			StatusColor: item.MauTrangThai,
			StatusCode:  item.MaTrangThai,
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write floor monitoring response: %v", err)
	}
}

func (h *Handler) ListBuildingAreas(w http.ResponseWriter, r *http.Request) {
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

	filter, err := parseBuildingAreaFilter(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	page, err := h.service.ListBuildingAreas(r.Context(), principal.IDToaNha, filter)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if page == nil {
		httpx.WriteError(w, errors.New("building area service returned nil page"))
		return
	}

	items := make([]buildingAreaResponse, len(page.Items))
	for i, item := range page.Items {
		items[i] = buildingAreaResponse{
			AreaID:          item.IDKhuVuc,
			AreaName:        item.TenKhuVuc,
			AreaDescription: item.MoTaKhuVuc,
			FloorID:         item.IDTang,
			FloorNumber:     item.SoTang,
			AreaTypeID:      item.IDLoaiKhuVuc,
			AreaTypeName:    item.TenLoaiKhuVuc,
			ResidentCount:   item.SoLuongCuDan,
			StatusColor:     item.MauTrangThai,
			StatusCode:      item.MaTrangThai,
		}
	}
	floors := make([]buildingFloorResponse, len(page.Floors))
	for i, floor := range page.Floors {
		floors[i] = buildingFloorResponse{
			FloorID:     floor.IDTang,
			FloorNumber: floor.SoTang,
		}
	}
	areaTypes := make([]buildingAreaTypeResponse, len(page.AreaTypes))
	for i, areaType := range page.AreaTypes {
		areaTypes[i] = buildingAreaTypeResponse{
			AreaTypeID:   areaType.IDLoaiKhuVuc,
			AreaTypeName: areaType.TenLoaiKhuVuc,
		}
	}

	response := buildingAreaListResponse{
		Items:      items,
		TotalCount: page.TotalCount,
		Floors:     floors,
		AreaTypes:  areaTypes,
		Pagination: buildingAreaPagination{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.TotalCount,
		},
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write building area list response: %v", err)
	}
}

func toRoomMonitoringResponse(item RoomMonitoring) roomMonitoringResponse {
	return roomMonitoringResponse{
		FloorNumber:       item.SoTang,
		FloorID:           item.IDTang,
		AreaID:            item.IDKhuVuc,
		AreaName:          item.TenKhuVuc,
		AreaDescription:   item.MoTaKhuVuc,
		AreaTypeName:      item.TenLoaiKhuVuc,
		AlertLevel:        item.MucDoCanhBao,
		AlertStatus:       item.TrangThaiCanhBao,
		AlertDescription:  item.MoTaCanhBao,
		HasFireAlert:      item.CoCanhBao,
		HasHardwareError:  item.CoLoiPhanCung,
		HardwareStatus:    item.TrangThaiPhanCung,
		Temperature:       item.NhietDo,
		TemperatureUnit:   item.DonViNhietDo,
		SmokeDensity:      item.NongDoKhoi,
		SmokeUnit:         item.DonViKhoi,
		StatusDescription: item.MoTaTinhTrang,
	}
}

func parseMonitoringFilter(r *http.Request) (MonitoringFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"filter_so_tang":       {},
		"floor_offset":         {},
		"search_keyword":       {},
		"filter_canh_bao_chay": {},
		"filter_loi_phan_cung": {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return MonitoringFilter{}, invalidMonitoringQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return MonitoringFilter{}, invalidMonitoringQuery(key, "query parameter must be specified once")
		}
	}

	filter := MonitoringFilter{}
	if values, ok := query["filter_so_tang"]; ok {
		value, err := strconv.Atoi(values[0])
		if err != nil {
			return MonitoringFilter{}, invalidMonitoringQuery("filter_so_tang", "filter_so_tang must be an integer")
		}
		filter.FloorNumber = &value
	}
	if values, ok := query["floor_offset"]; ok {
		value, err := strconv.Atoi(values[0])
		if err != nil || value < 0 {
			return MonitoringFilter{}, invalidMonitoringQuery("floor_offset", "floor_offset must be a non-negative integer")
		}
		filter.FloorOffset = value
	}
	if value := strings.TrimSpace(query.Get("search_keyword")); value != "" {
		filter.SearchKeyword = &value
	}
	if values, ok := query["filter_canh_bao_chay"]; ok {
		value, err := strconv.ParseBool(values[0])
		if err != nil {
			return MonitoringFilter{}, invalidMonitoringQuery("filter_canh_bao_chay", "filter_canh_bao_chay must be true or false")
		}
		filter.FireAlertOnly = &value
	}
	if values, ok := query["filter_loi_phan_cung"]; ok {
		value, err := strconv.ParseBool(values[0])
		if err != nil {
			return MonitoringFilter{}, invalidMonitoringQuery("filter_loi_phan_cung", "filter_loi_phan_cung must be true or false")
		}
		filter.HardwareErrorOnly = &value
	}
	return filter, nil
}

func parseBuildingAreaFilter(r *http.Request) (BuildingAreaFilter, error) {
	query := r.URL.Query()
	allowed := map[string]struct{}{
		"limit":               {},
		"offset":              {},
		"id_loai_khu_vuc":     {},
		"loai_khu_vuc":        {},
		"filter_loai_khu_vuc": {},
		"so_tang":             {},
		"filter_so_tang":      {},
		"ma_trang_thai":       {},
		"trang_thai":          {},
		"search":              {},
		"search_keyword":      {},
		"ten_khu_vuc":         {},
	}
	for key, values := range query {
		if _, ok := allowed[key]; !ok {
			return BuildingAreaFilter{}, invalidMonitoringQuery(key, "unknown query parameter")
		}
		if len(values) != 1 {
			return BuildingAreaFilter{}, invalidMonitoringQuery(key, "query parameter must be specified once")
		}
	}

	filter := BuildingAreaFilter{
		Limit: DefaultBuildingAreaPageSize,
	}
	if values, ok := query["limit"]; ok {
		value, err := strconv.Atoi(values[0])
		if err != nil || value < 1 || value > MaxBuildingAreaPageSize {
			return BuildingAreaFilter{}, invalidMonitoringQuery("limit", "limit must be an integer between 1 and 100")
		}
		filter.Limit = value
	}
	if values, ok := query["offset"]; ok {
		value, err := strconv.ParseInt(values[0], 10, 64)
		if err != nil || value < 0 {
			return BuildingAreaFilter{}, invalidMonitoringQuery("offset", "offset must be a non-negative integer")
		}
		filter.Offset = value
	}

	for _, key := range []string{"id_loai_khu_vuc", "loai_khu_vuc", "filter_loai_khu_vuc"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			filter.AreaTypeID = &value
			break
		}
	}
	for _, key := range []string{"so_tang", "filter_so_tang"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			number, err := strconv.Atoi(value)
			if err != nil {
				return BuildingAreaFilter{}, invalidMonitoringQuery(key, key+" must be an integer")
			}
			filter.FloorNumber = &number
			break
		}
	}
	for _, key := range []string{"ma_trang_thai", "trang_thai"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			filter.StatusCode = &value
			break
		}
	}
	for _, key := range []string{"search", "search_keyword", "ten_khu_vuc"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			filter.SearchKeyword = &value
			break
		}
	}
	return filter, nil
}

func invalidMonitoringQuery(field, message string) error {
	return apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		message,
		map[string]any{"field": field},
	)
}
