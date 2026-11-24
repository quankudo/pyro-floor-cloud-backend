package tang

import (
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

var ErrBuildingNotFound = apperror.New(
	http.StatusNotFound,
	apperror.AppErrorNotFound,
	"building not found",
	nil,
)
