package khuvuc

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

var ErrFloorNotFound = apperror.New(
	http.StatusNotFound,
	apperror.AppErrorNotFound,
	"floor not found",
	nil,
)
