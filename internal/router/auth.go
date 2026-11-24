package router

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/module/auth"
)

const (
	authLoginRoute         = "POST /api/v1/auth/login"
	authOwnerLoginRoute    = "POST /api/v1/auth/login/owner"
	authManagerLoginRoute  = "POST /api/v1/auth/login/manager"
	authGuardLoginRoute    = "POST /api/v1/auth/login/guard"
	authResidentLoginRoute = "POST /api/v1/auth/login/resident"
	authLogoutRoute        = "POST /api/v1/auth/logout"
	authRegisterRoute      = "POST /api/v1/auth/register"
)

func registerAuthRoutes(mux *http.ServeMux, handler *auth.AuthHandler) {
	mux.HandleFunc(authLoginRoute, handler.Login)
	mux.HandleFunc(authOwnerLoginRoute, handler.LoginOwner)
	mux.HandleFunc(authManagerLoginRoute, handler.LoginManager)
	mux.HandleFunc(authGuardLoginRoute, handler.LoginGuard)
	mux.HandleFunc(authResidentLoginRoute, handler.LoginResident)
	mux.HandleFunc(authRegisterRoute, handler.Register)
	mux.HandleFunc(authLogoutRoute, handler.Logout)
}
