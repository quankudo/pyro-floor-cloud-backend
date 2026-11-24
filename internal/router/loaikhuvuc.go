package router

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/middleware"
	"github.com/pyrofloor/cloud-backend/internal/module/loaikhuvuc"
)

const listLoaiKhuVucByModelTypeRoute = "GET /loai-mo-hinh/{id}/loai-khu-vuc"

func registerLoaiKhuVucRoutes(
	mux *http.ServeMux,
	handler *loaikhuvuc.Handler,
	authentication *middleware.Auth,
) {
	mux.Handle(
		listLoaiKhuVucByModelTypeRoute,
		authentication.RequireAuth(http.HandlerFunc(handler.ListByModelTypeID)),
	)
}
