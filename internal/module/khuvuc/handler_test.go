package khuvuc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeUseCase struct {
	called  bool
	floorID string
	items   []KhuVuc
	err     error
}

func (s *fakeUseCase) ListByFloorID(_ context.Context, floorID string) ([]KhuVuc, error) {
	s.called = true
	s.floorID = floorID
	return s.items, s.err
}

func (s *fakeUseCase) ListShortByFloorID(_ context.Context, floorID string) ([]KhuVucShort, error) {
	s.called = true
	s.floorID = floorID
	shortItems := make([]KhuVucShort, len(s.items))
	for i, item := range s.items {
		shortItems[i] = KhuVucShort{
			IDKhuVuc:     item.IDKhuVuc,
			TenKhuVuc:    item.TenKhuVuc,
			IDLoaiKhuVuc: item.IDLoaiKhuVuc,
		}
	}
	return shortItems, s.err
}

func (s *fakeUseCase) ListMonitoring(
	_ context.Context,
	_ string,
	_ MonitoringFilter,
) ([]RoomMonitoring, error) {
	return nil, nil
}

func (s *fakeUseCase) ListFloorMonitoring(
	_ context.Context,
	_ string,
) ([]FloorMonitoring, error) {
	return nil, nil
}

func (s *fakeUseCase) ListBuildingAreas(
	_ context.Context,
	_ string,
	_ BuildingAreaFilter,
) (*BuildingAreaPage, error) {
	return nil, nil
}

func TestHandlerListByFloorIDReturnsAreaInformation(t *testing.T) {
	service := &fakeUseCase{
		items: []KhuVuc{{
			IDKhuVuc:     "area-1",
			IDTang:       "floor-1",
			IDLoaiKhuVuc: "area-type-1",
			TenKhuVuc:    "Living room",
			MoTa:         "Main living area",
		}},
	}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/tang/floor-1/khu-vuc", nil)
	request.SetPathValue("id", "550e8400-e29b-41d4-a716-446655440000")
	recorder := httptest.NewRecorder()

	handler.ListByFloorID(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !service.called || service.floorID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("service call = %+v", service)
	}
	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			IDKhuVuc     string `json:"id_khu_vuc"`
			IDTang       string `json:"id_tang"`
			IDLoaiKhuVuc string `json:"id_loai_khu_vuc"`
			TenKhuVuc    string `json:"ten_khu_vuc"`
			MoTa         string `json:"mo_ta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Data) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	item := response.Data[0]
	if item.IDKhuVuc != "area-1" || item.IDTang != "floor-1" ||
		item.IDLoaiKhuVuc != "area-type-1" || item.TenKhuVuc != "Living room" ||
		item.MoTa != "Main living area" {
		t.Fatalf("area response = %+v", item)
	}
}

func TestHandlerListByFloorIDReturnsEmptyArray(t *testing.T) {
	handler, err := NewHandler(&fakeUseCase{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/tang/floor-id/khu-vuc", nil)
	request.SetPathValue("id", "550e8400-e29b-41d4-a716-446655440000")
	recorder := httptest.NewRecorder()

	handler.ListByFloorID(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || string(response.Data) != "[]" {
		t.Fatalf("response data = %s, want []", response.Data)
	}
}

func TestHandlerListByFloorIDRejectsQueryParameters(t *testing.T) {
	service := &fakeUseCase{}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/tang/id/khu-vuc?limit=10", nil)
	request.SetPathValue("id", "550e8400-e29b-41d4-a716-446655440000")
	recorder := httptest.NewRecorder()

	handler.ListByFloorID(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if service.called {
		t.Fatal("service called when query parameters are provided")
	}
}
