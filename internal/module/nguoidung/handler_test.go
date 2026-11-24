package nguoidung

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pyrofloor/cloud-backend/internal/authctx"
)

func TestHasResidentPaginationOnlyQuery(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "no query", url: "/api/v1/cu-dan", want: true},
		{name: "limit only", url: "/api/v1/cu-dan?limit=20", want: true},
		{name: "limit and offset", url: "/api/v1/cu-dan?limit=20&offset=0", want: true},
		{name: "non pagination offset", url: "/api/v1/cu-dan?offset=20", want: true},
		{name: "search query", url: "/api/v1/cu-dan?search=quan", want: false},
		{name: "floor filter", url: "/api/v1/cu-dan?id_tang=550e8400-e29b-41d4-a716-446655440001", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if got := hasResidentPaginationOnlyQuery(request); got != tt.want {
				t.Fatalf("hasResidentPaginationOnlyQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeBatchUserUseCase struct {
	called     bool
	actorID    string
	actorRole  string
	buildingID string
	requests   []BatchUserRequest
	created    []CreatedUser
	err        error
}

func (s *fakeBatchUserUseCase) CreateBatch(
	_ context.Context,
	actorID, actorRole string,
	buildingID string,
	requests []BatchUserRequest,
) ([]CreatedUser, error) {
	s.called = true
	s.actorID = actorID
	s.actorRole = actorRole
	s.buildingID = buildingID
	s.requests = requests
	return s.created, s.err
}

func TestNguoiDungHandlerCreateBatchDecodesUsersAndReturnsCreatedUsers(t *testing.T) {
	service := &fakeBatchUserUseCase{
		created: []CreatedUser{{
			NguoiDung: NguoiDung{
				IDNguoiDung:  "550e8400-e29b-41d4-a716-446655440010",
				IDVaiTro:     "550e8400-e29b-41d4-a716-446655440001",
				TenNguoiDung: "Resident One",
				Email:        "resident@example.com",
				TrangThai:    HoatDong,
			},
			KyHieuVaiTro: RoleCodeResident,
		}},
	}
	handler, err := NewNguoiDungHandler(service)
	if err != nil {
		t.Fatalf("NewNguoiDungHandler() error = %v", err)
	}

	body := `{"id_toa_nha":"550e8400-e29b-41d4-a716-446655440030","users":[{"ten_nguoi_dung":"Resident One","email":"resident@example.com","mat_khau":"password-123","ky_hieu_vai_tro":"CU_DAN","id_khu_vuc":"550e8400-e29b-41d4-a716-446655440020"}]}`
	request := httptest.NewRequest(http.MethodPost, "/nguoi-dung/batch", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(authctx.WithPrincipal(request.Context(), authctx.Principal{
		UserID: "550e8400-e29b-41d4-a716-446655440002",
		RoleID: RoleCodeOwner,
	}))
	recorder := httptest.NewRecorder()

	handler.CreateBatch(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if !service.called || service.actorID != "550e8400-e29b-41d4-a716-446655440002" ||
		service.actorRole != RoleCodeOwner ||
		service.buildingID != "550e8400-e29b-41d4-a716-446655440030" ||
		len(service.requests) != 1 {
		t.Fatalf("service call = %+v", service)
	}
	if service.requests[0].IDKhuVuc != "550e8400-e29b-41d4-a716-446655440020" ||
		service.requests[0].KyHieuVaiTro != RoleCodeResident {
		t.Fatalf("decoded user request = %+v", service.requests[0])
	}

	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			IDNguoiDung  string `json:"id_nguoi_dung"`
			KyHieuVaiTro string `json:"ky_hieu_vai_tro"`
			TrangThai    string `json:"trang_thai"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || len(response.Data) != 1 ||
		response.Data[0].IDNguoiDung != service.created[0].IDNguoiDung ||
		response.Data[0].KyHieuVaiTro != RoleCodeResident ||
		response.Data[0].TrangThai != string(HoatDong) {
		t.Fatalf("unexpected response: %+v", response)
	}
	if strings.Contains(recorder.Body.String(), "trang_thai_phan_cong") {
		t.Fatal("response exposed an aggregate assignment status")
	}
	if strings.Contains(recorder.Body.String(), "password") {
		t.Fatal("response exposed password field")
	}
}

func TestNguoiDungHandlerCreateBatchRequiresAuthenticatedPrincipal(t *testing.T) {
	service := &fakeBatchUserUseCase{}
	handler, err := NewNguoiDungHandler(service)
	if err != nil {
		t.Fatalf("NewNguoiDungHandler() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/nguoi-dung/batch", strings.NewReader(`{"users":[]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.CreateBatch(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if service.called {
		t.Fatal("service called without an authenticated principal")
	}
}

func TestNguoiDungHandlerCreateBatchRejectsInvalidRequestBeforeService(t *testing.T) {
	for _, tt := range []struct {
		name        string
		contentType string
		body        string
		query       string
		status      int
	}{
		{name: "unknown field", contentType: "application/json", body: `{"users":[],"unexpected":true}`, status: http.StatusBadRequest},
		{name: "unsupported content type", contentType: "text/plain", body: `{}`, status: http.StatusUnsupportedMediaType},
		{name: "query not supported", contentType: "application/json", body: `{"users":[]}`, query: "?limit=1", status: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeBatchUserUseCase{}
			handler, err := NewNguoiDungHandler(service)
			if err != nil {
				t.Fatalf("NewNguoiDungHandler() error = %v", err)
			}
			request := httptest.NewRequest(
				http.MethodPost,
				"/nguoi-dung/batch"+tt.query,
				strings.NewReader(tt.body),
			)
			request.Header.Set("Content-Type", tt.contentType)
			request = request.WithContext(authctx.WithPrincipal(request.Context(), authctx.Principal{
				UserID: "550e8400-e29b-41d4-a716-446655440002",
				RoleID: RoleCodeAdmin,
			}))
			recorder := httptest.NewRecorder()

			handler.CreateBatch(recorder, request)

			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.status)
			}
			if service.called {
				t.Fatal("service called with invalid request")
			}
		})
	}
}

func TestNguoiDungHandlerRejectsBuildingIDInsideManagerAssignment(t *testing.T) {
	service := &fakeBatchUserUseCase{}
	handler, err := NewNguoiDungHandler(service)
	if err != nil {
		t.Fatalf("NewNguoiDungHandler() error = %v", err)
	}
	body := `{"id_toa_nha":"550e8400-e29b-41d4-a716-446655440030","users":[{"ten_nguoi_dung":"Manager One","email":"manager@example.com","mat_khau":"password-123","ky_hieu_vai_tro":"QUAN_LY","phan_cong":{"id_toa_nha":"550e8400-e29b-41d4-a716-446655440030"}}]}`
	request := httptest.NewRequest(http.MethodPost, "/nguoi-dung/batch", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(authctx.WithPrincipal(request.Context(), authctx.Principal{
		UserID: "550e8400-e29b-41d4-a716-446655440002",
		RoleID: RoleCodeAdmin,
	}))
	recorder := httptest.NewRecorder()

	handler.CreateBatch(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if service.called {
		t.Fatal("service called with redundant nested building ID")
	}
}
