package auth

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

var (
	ErrInvalidLogin = apperror.New(
		http.StatusUnauthorized,
		apperror.AppErrorUnauthenticated,
		"email or password is incorrect",
		nil,
	)
	ErrAccountDeleted = apperror.New(
		http.StatusForbidden,
		apperror.AppErrorForbidden,
		"account is unavailable",
		nil,
	)
	ErrEmailNotVerified = apperror.New(
		http.StatusForbidden,
		apperror.AppErrorForbidden,
		"email is not verified",
		nil,
	)
)
