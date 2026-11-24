package toanha

import (
	"errors"
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

var (
	ErrToaNhaNotFound    = errors.New("toa nha not found")
	ErrBuildingForbidden = apperror.New(
		http.StatusForbidden,
		apperror.AppErrorForbidden,
		"building is not owned by the authenticated owner",
		nil,
	)
)
