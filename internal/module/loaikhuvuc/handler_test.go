package loaikhuvuc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeUseCase struct {
	called      bool
	modelTypeID string
	items       []LoaiKhuVuc
	err         error
}

func (s *fakeUseCase) ListByModelTypeID(
	_ context.Context,
	modelTypeID string,
) ([]LoaiKhuVuc, error) {
	s.called = true
	s.modelTypeID = modelTypeID
	return s.items, s.err
}

func TestHandlerListByModelTypeIDReturnsAllFields(t *testing.T) {
	service := &fakeUseCase{items: []LoaiKhuVuc{{
		IDLoaiKhuVuc:         "area-type-1",
		IDLoaiMoHinh:         "model-1",
		TenLoaiKhuVuc:        "Bedroom",
		ThoiGianChoXacNhan:   45,
		MucDoFallbackMacDinh: "DANGER",
		SuDungGiaTriCamBien:  true,
		MoTa:                 "Sleeping area",
	}}}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/loai-mo-hinh/model-1/loai-khu-vuc", nil)
	request.SetPathValue("id", "550e8400-e29b-41d4-a716-446655440000")
	recorder := httptest.NewRecorder()

	handler.ListByModelTypeID(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !service.called || service.modelTypeID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("service call = %+v", service)
	}
	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			IDLoaiKhuVuc         string `json:"id_loai_khu_vuc"`
			IDLoaiMoHinh         string `json:"id_loai_mo_hinh"`
			TenLoaiKhuVuc        string `json:"ten_loai_khu_vuc"`
			ThoiGianChoXacNhan   int    `json:"thoi_gian_cho_xac_nhan"`
			MucDoFallbackMacDinh string `json:"muc_do_fallback_mac_dinh"`
			SuDungGiaTriCamBien  bool   `json:"su_dung_gia_tri_cam_bien"`
			MoTa                 string `json:"mo_ta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Data) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	item := response.Data[0]
	if item.IDLoaiKhuVuc != "area-type-1" ||
		item.IDLoaiMoHinh != "model-1" ||
		item.TenLoaiKhuVuc != "Bedroom" ||
		item.ThoiGianChoXacNhan != 45 ||
		item.MucDoFallbackMacDinh != "DANGER" ||
		!item.SuDungGiaTriCamBien ||
		item.MoTa != "Sleeping area" {
		t.Fatalf("area type response = %+v", item)
	}
}

func TestHandlerListByModelTypeIDReturnsEmptyArray(t *testing.T) {
	handler, err := NewHandler(&fakeUseCase{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/loai-mo-hinh/id/loai-khu-vuc", nil)
	request.SetPathValue("id", "550e8400-e29b-41d4-a716-446655440000")
	recorder := httptest.NewRecorder()

	handler.ListByModelTypeID(recorder, request)

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

func TestHandlerListByModelTypeIDRejectsQueryParameters(t *testing.T) {
	service := &fakeUseCase{}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/loai-mo-hinh/id/loai-khu-vuc?limit=10", nil)
	request.SetPathValue("id", "550e8400-e29b-41d4-a716-446655440000")
	recorder := httptest.NewRecorder()

	handler.ListByModelTypeID(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if service.called {
		t.Fatal("service called when query parameters are provided")
	}
}
