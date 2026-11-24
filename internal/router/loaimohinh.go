package router

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/middleware"
	"github.com/pyrofloor/cloud-backend/internal/module/loaimohinh"
)

const listLoaiMoHinhRoute = "GET /loai-mo-hinh"

func registerLoaiMoHinhRoutes(
	mux *http.ServeMux,
	handler *loaimohinh.Handler,
	authentication *middleware.Auth,
) {
	mux.Handle(
		listLoaiMoHinhRoute,
		authentication.RequireAuth(http.HandlerFunc(handler.List)),
	)
}
