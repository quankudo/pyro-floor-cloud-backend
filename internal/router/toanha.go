package router

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/middleware"
	"github.com/pyrofloor/cloud-backend/internal/module/khuvuc"
	"github.com/pyrofloor/cloud-backend/internal/module/tang"
	"github.com/pyrofloor/cloud-backend/internal/module/toanha"
)

const (
	listToaNhaRoute          = "GET /toanha"
	getByIDRoute             = "GET /toanha/{id}"
	listFloorsRoute          = "GET /api/v1/tang"
	listAreasRoute           = "GET /tang/{id}/khu-vuc"
	listShortDataAreasRoute  = "GET /api/v1/tang/{id}/khu-vuc/short-data"
	listBuildingAreasRoute   = "GET /api/v1/khu-vuc"
	listRoomMonitoringRoute  = "GET /api/v1/khu-vuc/monitoring"
	listFloorMonitoringRoute = "GET /api/v1/khu-vuc/monitoring/floors"

	adminRoleID = "QUAN_TRI_VIEN"
)

func registerToaNhaRoutes(
	mux *http.ServeMux,
	handler *toanha.ToaNhaHandler,
	authentication *middleware.Auth,
	requireAdmin func(http.Handler) http.Handler,
) {
	mux.Handle(
		listToaNhaRoute,
		authentication.RequireAuth(requireAdmin(http.HandlerFunc(handler.List))),
	)
	mux.Handle(
		getByIDRoute,
		authentication.RequireAuth(http.HandlerFunc(handler.GetByID)),
	)
}

func registerFloorRoutes(
	mux *http.ServeMux,
	handler *tang.Handler,
	authentication *middleware.Auth,
) {
	mux.Handle(
		listFloorsRoute,
		authentication.RequireAuth(http.HandlerFunc(handler.ListByBuildingID)),
	)
}

func registerAreaRoutes(
	mux *http.ServeMux,
	handler *khuvuc.Handler,
	authentication *middleware.Auth,
	requireAreaViewer func(http.Handler) http.Handler,
) {
	mux.Handle(
		listAreasRoute,
		authentication.RequireAuth(http.HandlerFunc(handler.ListByFloorID)),
	)

	mux.Handle(
		listShortDataAreasRoute,
		authentication.RequireAuth(http.HandlerFunc(handler.ListShortDataByFloorID)),
	)

	mux.Handle(
		listBuildingAreasRoute,
		authentication.RequireAuth(
			requireAreaViewer(http.HandlerFunc(handler.ListBuildingAreas)),
		),
	)
	mux.Handle(
		listRoomMonitoringRoute,
		authentication.RequireAuth(
			requireAreaViewer(http.HandlerFunc(handler.ListMonitoring)),
		),
	)
	mux.Handle(
		listFloorMonitoringRoute,
		authentication.RequireAuth(
			requireAreaViewer(http.HandlerFunc(handler.ListFloorMonitoring)),
		),
	)
}
