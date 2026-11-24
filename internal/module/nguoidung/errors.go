package nguoidung

import (
	"errors"
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrBatchSizeInvalid = apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		"users must contain between 1 and 25 entries",
		map[string]any{"field": "users"},
	)
	ErrRoleNotFound = apperror.New(
		http.StatusBadRequest,
		apperror.AppErrorInvalidArgument,
		"requested role does not exist",
		nil,
	)
	ErrEmailExists = apperror.New(
		http.StatusConflict,
		apperror.AppErrorConflict,
		"email already exists",
		nil,
	)
	ErrRoleNotAllowed = apperror.New(
		http.StatusForbidden,
		apperror.AppErrorForbidden,
		"only QUAN_LY and CU_DAN users can be added as building members",
		nil,
	)
	ErrExistingUserRoleMismatch = apperror.New(
		http.StatusConflict,
		apperror.AppErrorConflict,
		"existing user has a different global role; the role was not changed",
		nil,
	)
)
