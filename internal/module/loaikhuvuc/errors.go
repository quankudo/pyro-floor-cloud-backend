package loaikhuvuc

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

var ErrModelTypeNotFound = apperror.New(
	http.StatusNotFound,
	apperror.AppErrorNotFound,
	"model type not found",
	nil,
)
