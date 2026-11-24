package tang

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/authctx"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

type UseCase interface {
	ListByBuildingID(
		ctx context.Context,
		buildingID string,
	) ([]FloorsResponse, error)
}

type Handler struct {
	service UseCase
}

func NewHandler(service UseCase) (*Handler, error) {
	if service == nil {
		return nil, errors.New("floor service is required")
	}
	return &Handler{service: service}, nil
}

func (h *Handler) ListByBuildingID(w http.ResponseWriter, r *http.Request) {
	principal, ok := authctx.PrincipalFromContext(r.Context())
	if !ok || strings.TrimSpace(principal.IDToaNha) == "" {
		httpx.WriteError(w, apperror.New(
			http.StatusForbidden,
			apperror.AppErrorForbidden,
			"building scope is missing from authentication token",
			nil,
		))
		return
	}

	result, err := h.service.ListByBuildingID(r.Context(), principal.IDToaNha)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if result == nil {
		httpx.WriteError(w, errors.New("floor service returned nil page"))
		return
	}

	if err := httpx.WriteSuccess(w, http.StatusOK, result); err != nil {
		log.Printf("write list floors response: %v", err)
	}
}
