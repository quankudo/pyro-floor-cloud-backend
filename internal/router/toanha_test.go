package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pyrofloor/cloud-backend/internal/middleware"
	"github.com/pyrofloor/cloud-backend/internal/module/auth"
	"github.com/pyrofloor/cloud-backend/internal/module/khuvuc"
	"github.com/pyrofloor/cloud-backend/internal/module/loaikhuvuc"
	"github.com/pyrofloor/cloud-backend/internal/module/loaimohinh"
	"github.com/pyrofloor/cloud-backend/internal/module/nguoidung"
	"github.com/pyrofloor/cloud-backend/internal/module/tang"
	"github.com/pyrofloor/cloud-backend/internal/module/toanha"
)

type routeTokenVerifier struct {
	roleID string
	err    error
}

func (v routeTokenVerifier) VerifyAccessCookie(*http.Request) (*auth.TokenClaims, error) {
	if v.err != nil {
		return nil, v.err
	}
	return &auth.TokenClaims{
		UserID:   "user-1",
		RoleID:   v.roleID,
		IDToaNha: "550e8400-e29b-41d4-a716-446655440000",
	}, nil
}

type routeToaNhaService struct{}

func (routeToaNhaService) FindByID(context.Context, string) (*toanha.ToaNha, error) {
	return nil, nil
}

func (routeToaNhaService) List(context.Context, toanha.ListFilter) (*toanha.Page, error) {
	return &toanha.Page{Items: []toanha.ToaNha{}, Limit: toanha.DefaultPageSize}, nil
}

type routeFloorService struct{}

func (routeFloorService) ListByBuildingID(
	context.Context,
	string,
) ([]tang.FloorsResponse, error) {
	return []tang.FloorsResponse{}, nil
}

type routeAreaService struct{}

func (routeAreaService) ListByFloorID(context.Context, string) ([]khuvuc.KhuVuc, error) {
	return []khuvuc.KhuVuc{}, nil
}

func (routeAreaService) ListShortByFloorID(context.Context, string) ([]khuvuc.KhuVucShort, error) {
	return []khuvuc.KhuVucShort{}, nil
}

func (routeAreaService) ListMonitoring(
	context.Context,
	string,
	khuvuc.MonitoringFilter,
) ([]khuvuc.RoomMonitoring, error) {
	return []khuvuc.RoomMonitoring{}, nil
}

func (routeAreaService) ListFloorMonitoring(
	context.Context,
	string,
) ([]khuvuc.FloorMonitoring, error) {
	return []khuvuc.FloorMonitoring{}, nil
}

func (routeAreaService) ListBuildingAreas(
	context.Context,
	string,
	khuvuc.BuildingAreaFilter,
) (*khuvuc.BuildingAreaPage, error) {
	return &khuvuc.BuildingAreaPage{
		Items:     []khuvuc.BuildingArea{},
		Floors:    []khuvuc.BuildingFloor{},
		AreaTypes: []khuvuc.BuildingAreaType{},
		Limit:     khuvuc.DefaultBuildingAreaPageSize,
	}, nil
}

type routeLoaiMoHinhService struct{}

func (routeLoaiMoHinhService) List(context.Context) ([]loaimohinh.LoaiMoHinh, error) {
	return []loaimohinh.LoaiMoHinh{}, nil
}

type routeLoaiKhuVucService struct{}

func (routeLoaiKhuVucService) ListByModelTypeID(
	context.Context,
	string,
) ([]loaikhuvuc.LoaiKhuVuc, error) {
	return []loaikhuvuc.LoaiKhuVuc{}, nil
}

type routeBatchUserService struct{}

func (routeBatchUserService) CreateBatch(
	context.Context,
	string,
	string,
	string,
	[]nguoidung.BatchUserRequest,
) ([]nguoidung.CreatedUser, error) {
	return []nguoidung.CreatedUser{}, nil
}

func TestListToaNhaRouteRequiresAdminRole(t *testing.T) {
	for _, tt := range []struct {
		name           string
		roleID         string
		wantStatusCode int
	}{
		{name: "administrator", roleID: adminRoleID, wantStatusCode: http.StatusOK},
		{name: "other role", roleID: "CU_DAN", wantStatusCode: http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			authentication, err := middleware.NewAuth(routeTokenVerifier{roleID: tt.roleID})
			if err != nil {
				t.Fatalf("NewAuth() error = %v", err)
			}
			requireAdmin, err := authentication.RequireRoles(adminRoleID)
			if err != nil {
				t.Fatalf("RequireRoles() error = %v", err)
			}
			handler, err := toanha.NewToaNhaHandler(routeToaNhaService{})
			if err != nil {
				t.Fatalf("NewToaNhaHandler() error = %v", err)
			}

			mux := http.NewServeMux()
			registerToaNhaRoutes(mux, handler, authentication, requireAdmin)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/toanha", nil))

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestListFloorsRouteRequiresAuthentication(t *testing.T) {
	for _, tt := range []struct {
		name           string
		verifier       routeTokenVerifier
		wantStatusCode int
	}{
		{name: "authenticated", verifier: routeTokenVerifier{roleID: "CU_DAN"}, wantStatusCode: http.StatusOK},
		{name: "unauthenticated", verifier: routeTokenVerifier{err: errors.New("invalid token")}, wantStatusCode: http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			authentication, err := middleware.NewAuth(tt.verifier)
			if err != nil {
				t.Fatalf("NewAuth() error = %v", err)
			}
			handler, err := tang.NewHandler(routeFloorService{})
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}
			mux := http.NewServeMux()
			registerFloorRoutes(mux, handler, authentication)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/toanha/550e8400-e29b-41d4-a716-446655440000/tang",
				nil,
			)
			mux.ServeHTTP(recorder, request)
			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestListAreasRouteRequiresAuthentication(t *testing.T) {
	for _, tt := range []struct {
		name           string
		verifier       routeTokenVerifier
		wantStatusCode int
	}{
		{name: "authenticated", verifier: routeTokenVerifier{roleID: "CU_DAN"}, wantStatusCode: http.StatusOK},
		{name: "unauthenticated", verifier: routeTokenVerifier{err: errors.New("invalid token")}, wantStatusCode: http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			authentication, err := middleware.NewAuth(tt.verifier)
			if err != nil {
				t.Fatalf("NewAuth() error = %v", err)
			}
			handler, err := khuvuc.NewHandler(routeAreaService{})
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}
			mux := http.NewServeMux()
			registerAreaRoutes(mux, handler, authentication, func(next http.Handler) http.Handler {
				return next
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				"/tang/550e8400-e29b-41d4-a716-446655440000/khu-vuc",
				nil,
			)
			mux.ServeHTTP(recorder, request)
			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestListLoaiMoHinhRouteRequiresAuthentication(t *testing.T) {
	for _, tt := range []struct {
		name           string
		verifier       routeTokenVerifier
		wantStatusCode int
	}{
		{name: "authenticated", verifier: routeTokenVerifier{roleID: "CU_DAN"}, wantStatusCode: http.StatusOK},
		{name: "unauthenticated", verifier: routeTokenVerifier{err: errors.New("invalid token")}, wantStatusCode: http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			authentication, err := middleware.NewAuth(tt.verifier)
			if err != nil {
				t.Fatalf("NewAuth() error = %v", err)
			}
			handler, err := loaimohinh.NewHandler(routeLoaiMoHinhService{})
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}
			mux := http.NewServeMux()
			registerLoaiMoHinhRoutes(mux, handler, authentication)

			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/loai-mo-hinh", nil))
			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestListLoaiKhuVucRouteRequiresAuthentication(t *testing.T) {
	for _, tt := range []struct {
		name           string
		verifier       routeTokenVerifier
		wantStatusCode int
	}{
		{name: "authenticated", verifier: routeTokenVerifier{roleID: "CU_DAN"}, wantStatusCode: http.StatusOK},
		{name: "unauthenticated", verifier: routeTokenVerifier{err: errors.New("invalid token")}, wantStatusCode: http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			authentication, err := middleware.NewAuth(tt.verifier)
			if err != nil {
				t.Fatalf("NewAuth() error = %v", err)
			}
			handler, err := loaikhuvuc.NewHandler(routeLoaiKhuVucService{})
			if err != nil {
				t.Fatalf("NewHandler() error = %v", err)
			}
			mux := http.NewServeMux()
			registerLoaiKhuVucRoutes(mux, handler, authentication)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				"/loai-mo-hinh/550e8400-e29b-41d4-a716-446655440000/loai-khu-vuc",
				nil,
			)
			mux.ServeHTTP(recorder, request)
			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestCreateNguoiDungBatchRouteAllowsOnlyOwner(t *testing.T) {
	for _, tt := range []struct {
		roleID         string
		wantStatusCode int
	}{
		{roleID: nguoidung.RoleCodeAdmin, wantStatusCode: http.StatusUnsupportedMediaType},
		{roleID: nguoidung.RoleCodeOwner, wantStatusCode: http.StatusUnsupportedMediaType},
		{roleID: nguoidung.RoleCodeResident, wantStatusCode: http.StatusForbidden},
	} {
		t.Run(tt.roleID, func(t *testing.T) {
			authentication, err := middleware.NewAuth(routeTokenVerifier{roleID: tt.roleID})
			if err != nil {
				t.Fatalf("NewAuth() error = %v", err)
			}
			handler, err := nguoidung.NewNguoiDungHandler(routeBatchUserService{})
			if err != nil {
				t.Fatalf("NewNguoiDungHandler() error = %v", err)
			}
			requireUserAdmin, err := authentication.RequireRoles(
				nguoidung.RoleCodeAdmin,
				nguoidung.RoleCodeOwner,
			)
			if err != nil {
				t.Fatalf("RequireRoles() error = %v", err)
			}
			mux := http.NewServeMux()
			registerNguoiDungRoutes(mux, handler, authentication, requireUserAdmin, nil)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/nguoi-dung/batch", nil)
			mux.ServeHTTP(recorder, request)
			if recorder.Code != tt.wantStatusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatusCode)
			}
		})
	}
}
