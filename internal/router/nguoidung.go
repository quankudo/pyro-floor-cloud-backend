package router

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/middleware"
	"github.com/pyrofloor/cloud-backend/internal/module/nguoidung"
)

const createNguoiDungBatchRoute = "POST /nguoi-dung/batch"
const listResidentsRoute = "GET /api/v1/cu-dan"
const listSupervisorActorsRoute = "GET /api/v1/quan-ly-bao-ve"

func registerNguoiDungRoutes(
	mux *http.ServeMux,
	handler *nguoidung.NguoiDungHandler,
	authentication *middleware.Auth,
	requireUserAdmin func(http.Handler) http.Handler,
	requireResidentViewer func(http.Handler) http.Handler,
) {
	mux.Handle(
		createNguoiDungBatchRoute,
		authentication.RequireAuth(requireUserAdmin(http.HandlerFunc(handler.CreateBatch))),
	)
	if requireResidentViewer != nil {
		mux.Handle(
			listResidentsRoute,
			authentication.RequireAuth(requireResidentViewer(http.HandlerFunc(handler.ListResidents))),
		)
		mux.Handle(
			listSupervisorActorsRoute,
			authentication.RequireAuth(requireResidentViewer(http.HandlerFunc(handler.ListSupervisorActors))),
		)
	}
}
