package toanha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeToaNhaUseCase struct {
	filter    ListFilter
	page      *Page
	listCalls int
}

func (s *fakeToaNhaUseCase) FindByID(context.Context, string) (*ToaNha, error) {
	return nil, nil
}

func (s *fakeToaNhaUseCase) List(_ context.Context, filter ListFilter) (*Page, error) {
	s.listCalls++
	s.filter = filter
	return s.page, nil
}

func TestToaNhaHandlerListParsesFiltersAndPagination(t *testing.T) {
	service := &fakeToaNhaUseCase{
		page: &Page{
			Items: []ToaNha{{
				IDToaNha:     "building-1",
				IDLoaiMoHinh: "model-1",
				TenToaNha:    "Chung cu A",
				NgayTao:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
				TrangThai:    "HoatDong",
			}},
			Total:  12,
			Limit:  5,
			Offset: 10,
		},
	}
	handler, err := NewToaNhaHandler(service)
	if err != nil {
		t.Fatalf("NewToaNhaHandler() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/toanha?limit=5&offset=10&trang_thai=HoatDong&id_loai_mo_hinh=550e8400-e29b-41d4-a716-446655440000&ten_toa_nha=Chung+cu",
		nil,
	)
	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if service.listCalls != 1 {
		t.Fatalf("List() calls = %d, want 1", service.listCalls)
	}
	wantFilter := ListFilter{
		Limit:        5,
		Offset:       10,
		TrangThai:    "HoatDong",
		IDLoaiMoHinh: "550e8400-e29b-41d4-a716-446655440000",
		TenToaNha:    "Chung cu",
	}
	if service.filter != wantFilter {
		t.Fatalf("service filter = %+v, want %+v", service.filter, wantFilter)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				IDToaNha string `json:"id_toa_nha"`
			} `json:"items"`
			Pagination paginationResponse `json:"pagination"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Data.Items) != 1 ||
		response.Data.Items[0].IDToaNha != "building-1" ||
		response.Data.Pagination != (paginationResponse{Limit: 5, Offset: 10, Total: 12}) {
		t.Fatalf("unexpected list response: %+v", response)
	}
}

func TestToaNhaHandlerListUsesDefaultPagination(t *testing.T) {
	service := &fakeToaNhaUseCase{page: &Page{Items: []ToaNha{}, Limit: DefaultPageSize}}
	handler, err := NewToaNhaHandler(service)
	if err != nil {
		t.Fatalf("NewToaNhaHandler() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.List(recorder, httptest.NewRequest(http.MethodGet, "/toanha", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if service.filter.Limit != DefaultPageSize || service.filter.Offset != 0 {
		t.Fatalf("default filter = %+v, want limit %d and offset 0", service.filter, DefaultPageSize)
	}
}

func TestToaNhaHandlerListRejectsInvalidQuery(t *testing.T) {
	for _, path := range []string{
		"/toanha?limit=0",
		"/toanha?offset=-1",
		"/toanha?limit=1&limit=2",
		"/toanha?unknown=value",
	} {
		t.Run(path, func(t *testing.T) {
			service := &fakeToaNhaUseCase{}
			handler, err := NewToaNhaHandler(service)
			if err != nil {
				t.Fatalf("NewToaNhaHandler() error = %v", err)
			}

			recorder := httptest.NewRecorder()
			handler.List(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
			}
			if service.listCalls != 0 {
				t.Fatal("service List() called for invalid query")
			}
		})
	}
}
