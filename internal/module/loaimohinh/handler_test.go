package loaimohinh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeUseCase struct {
	called bool
	items  []LoaiMoHinh
	err    error
}

func (s *fakeUseCase) List(context.Context) ([]LoaiMoHinh, error) {
	s.called = true
	return s.items, s.err
}

func TestHandlerListReturnsAllModelTypeFields(t *testing.T) {
	service := &fakeUseCase{items: []LoaiMoHinh{{
		IDLoaiMoHinh:  "model-1",
		TenLoaiMoHinh: "Apartment",
		MoTa:          "Residential apartment building",
	}}}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.List(recorder, httptest.NewRequest(http.MethodGet, "/loai-mo-hinh", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !service.called {
		t.Fatal("service List() was not called")
	}

	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			IDLoaiMoHinh  string `json:"id_loai_mo_hinh"`
			TenLoaiMoHinh string `json:"ten_loai_mo_hinh"`
			MoTa          string `json:"mo_ta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Data) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	item := response.Data[0]
	if item.IDLoaiMoHinh != "model-1" ||
		item.TenLoaiMoHinh != "Apartment" ||
		item.MoTa != "Residential apartment building" {
		t.Fatalf("model type response = %+v", item)
	}
}

func TestHandlerListReturnsEmptyArray(t *testing.T) {
	handler, err := NewHandler(&fakeUseCase{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.List(recorder, httptest.NewRequest(http.MethodGet, "/loai-mo-hinh", nil))

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

func TestHandlerListRejectsQueryParameters(t *testing.T) {
	service := &fakeUseCase{}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.List(recorder, httptest.NewRequest(http.MethodGet, "/loai-mo-hinh?limit=10", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if service.called {
		t.Fatal("service called when query parameters are provided")
	}
}
