package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/authctx"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
	"github.com/pyrofloor/cloud-backend/internal/module/auth"
)

type AccessTokenVerifier interface {
	VerifyAccessCookie(r *http.Request) (*auth.TokenClaims, error)
}

type Principal = authctx.Principal

type Auth struct {
	tokenVerifier AccessTokenVerifier
}

func NewAuth(tokenVerifier AccessTokenVerifier) (*Auth, error) {
	if tokenVerifier == nil {
		return nil, errors.New("access token verifier is required")
	}

	return &Auth{tokenVerifier: tokenVerifier}, nil
}

func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := a.tokenVerifier.VerifyAccessCookie(r)
		if err != nil || claims == nil ||
			strings.TrimSpace(claims.UserID) == "" ||
			strings.TrimSpace(claims.RoleID) == "" {
			httpx.WriteError(w, apperror.New(
				http.StatusUnauthorized,
				apperror.AppErrorUnauthenticated,
				"authentication is required",
				nil,
			))
			return
		}

		principal := Principal{
			UserID:   claims.UserID,
			RoleID:   claims.RoleID,
			IDToaNha: claims.IDToaNha,
			IDKhuVuc: claims.IDKhuVuc,
		}
		ctx := authctx.WithPrincipal(r.Context(), principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Auth) RequireRoles(roleIDs ...string) (func(http.Handler) http.Handler, error) {
	if len(roleIDs) == 0 {
		return nil, errors.New("at least one role ID is required")
	}

	allowedRoles := make(map[string]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			return nil, errors.New("role IDs must not be empty")
		}
		allowedRoles[roleID] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				httpx.WriteError(w, apperror.New(
					http.StatusUnauthorized,
					apperror.AppErrorUnauthenticated,
					"authentication is required",
					nil,
				))
				return
			}

			if _, allowed := allowedRoles[principal.RoleID]; !allowed {
				httpx.WriteError(w, apperror.New(
					http.StatusForbidden,
					apperror.AppErrorForbidden,
					"you do not have permission to access this resource",
					nil,
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	return authctx.PrincipalFromContext(ctx)
}
