package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
	"github.com/pyrofloor/cloud-backend/internal/middleware"
	"github.com/pyrofloor/cloud-backend/internal/module/auth"
	"github.com/pyrofloor/cloud-backend/internal/module/canhbao"
	"github.com/pyrofloor/cloud-backend/internal/module/khuvuc"
	"github.com/pyrofloor/cloud-backend/internal/module/loaikhuvuc"
	"github.com/pyrofloor/cloud-backend/internal/module/loaimohinh"
	"github.com/pyrofloor/cloud-backend/internal/module/nguoidung"
	"github.com/pyrofloor/cloud-backend/internal/module/phanconggiamsat"
	"github.com/pyrofloor/cloud-backend/internal/module/sos"
	"github.com/pyrofloor/cloud-backend/internal/module/tang"
	"github.com/pyrofloor/cloud-backend/internal/module/toanha"
)

const healthCheckTimeout = 2 * time.Second

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type Config struct {
	Health         HealthChecker
	Auth           *auth.AuthHandler
	NguoiDung      *nguoidung.NguoiDungHandler
	Assignments    *phanconggiamsat.Handler
	Alerts         *canhbao.Handler
	Areas          *khuvuc.Handler
	Floors         *tang.Handler
	LoaiKhuVuc     *loaikhuvuc.Handler
	LoaiMoHinh     *loaimohinh.Handler
	ToaNha         *toanha.ToaNhaHandler
	SOS            *sos.Handler
	Authentication *middleware.Auth
}

func New(cfg Config) (http.Handler, error) {
	if cfg.Health == nil {
		return nil, errors.New("health checker is required")
	}
	if cfg.Auth == nil {
		return nil, errors.New("auth handler is required")
	}
	if (cfg.ToaNha != nil || cfg.Floors != nil || cfg.Areas != nil || cfg.Assignments != nil || cfg.Alerts != nil ||
		cfg.LoaiMoHinh != nil || cfg.LoaiKhuVuc != nil || cfg.NguoiDung != nil || cfg.SOS != nil) &&
		cfg.Authentication == nil {
		return nil, errors.New("authentication middleware is required for protected routes")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler(cfg.Health))
	registerAuthRoutes(mux, cfg.Auth)
	if cfg.NguoiDung != nil {
		requireUserAdmin, err := cfg.Authentication.RequireRoles(
			nguoidung.RoleCodeAdmin,
			nguoidung.RoleCodeOwner,
		)
		if err != nil {
			return nil, fmt.Errorf("configure user creation roles: %w", err)
		}
		requireResidentViewer, err := cfg.Authentication.RequireRoles(
			nguoidung.RoleCodeOwner,
			nguoidung.RoleCodeManager,
			nguoidung.RoleCodeGuard,
		)
		if err != nil {
			return nil, fmt.Errorf("configure resident viewer roles: %w", err)
		}
		registerNguoiDungRoutes(
			mux,
			cfg.NguoiDung,
			cfg.Authentication,
			requireUserAdmin,
			requireResidentViewer,
		)
	}
	if cfg.ToaNha != nil {
		requireAdmin, err := cfg.Authentication.RequireRoles(adminRoleID)
		if err != nil {
			return nil, fmt.Errorf("configure toa nha admin role middleware: %w", err)
		}
		registerToaNhaRoutes(mux, cfg.ToaNha, cfg.Authentication, requireAdmin)
	}
	if cfg.Floors != nil {
		registerFloorRoutes(mux, cfg.Floors, cfg.Authentication)
	}
	if cfg.Areas != nil {
		requireAreaViewer, err := cfg.Authentication.RequireRoles(
			nguoidung.RoleCodeOwner,
			nguoidung.RoleCodeManager,
			nguoidung.RoleCodeGuard,
		)
		if err != nil {
			return nil, fmt.Errorf("configure area monitoring roles: %w", err)
		}
		registerAreaRoutes(mux, cfg.Areas, cfg.Authentication, requireAreaViewer)
	}
	if cfg.Assignments != nil {
		requireAssignmentViewer, err := cfg.Authentication.RequireRoles(
			nguoidung.RoleCodeOwner,
			nguoidung.RoleCodeManager,
			nguoidung.RoleCodeGuard,
		)
		if err != nil {
			return nil, fmt.Errorf("configure assignment viewer roles: %w", err)
		}
		mux.Handle(
			"GET /api/v1/phan-cong-giam-sat",
			cfg.Authentication.RequireAuth(
				requireAssignmentViewer(http.HandlerFunc(cfg.Assignments.ListAssignmentScopes)),
			),
		)
	}
	if cfg.Alerts != nil {
		requireAlertViewer, err := cfg.Authentication.RequireRoles(
			nguoidung.RoleCodeOwner,
			nguoidung.RoleCodeManager,
			nguoidung.RoleCodeGuard,
		)
		if err != nil {
			return nil, fmt.Errorf("configure alert viewer roles: %w", err)
		}
		mux.Handle(
			"GET /api/v1/canh-bao",
			cfg.Authentication.RequireAuth(
				requireAlertViewer(http.HandlerFunc(cfg.Alerts.List)),
			),
		)
		mux.Handle(
			"GET /api/v1/canh-bao/dang-xay-ra",
			cfg.Authentication.RequireAuth(
				requireAlertViewer(http.HandlerFunc(cfg.Alerts.ListActive)),
			),
		)
	}
	if cfg.SOS != nil {
		requireSOSViewer, err := cfg.Authentication.RequireRoles(
			nguoidung.RoleCodeOwner,
			nguoidung.RoleCodeManager,
			nguoidung.RoleCodeGuard,
		)
		if err != nil {
			return nil, fmt.Errorf("configure SOS viewer roles: %w", err)
		}
		mux.Handle(
			"GET /api/v1/sos/da-ket-thuc",
			cfg.Authentication.RequireAuth(
				requireSOSViewer(http.HandlerFunc(cfg.SOS.ListCompleted)),
			),
		)
		mux.Handle(
			"GET /api/v1/sos/dang-xay-ra",
			cfg.Authentication.RequireAuth(
				requireSOSViewer(http.HandlerFunc(cfg.SOS.ListActive)),
			),
		)
	}
	if cfg.LoaiMoHinh != nil {
		registerLoaiMoHinhRoutes(mux, cfg.LoaiMoHinh, cfg.Authentication)
	}
	if cfg.LoaiKhuVuc != nil {
		registerLoaiKhuVucRoutes(mux, cfg.LoaiKhuVuc, cfg.Authentication)
	}

	return mux, nil
}

func healthHandler(health HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		if err := health.Ping(ctx); err != nil {
			httpx.WriteError(w, apperror.Wrap(
				http.StatusServiceUnavailable,
				apperror.AppErrorUnavailable,
				"health check failed",
				nil,
				err,
			))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
