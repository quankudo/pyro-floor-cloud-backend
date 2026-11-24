package loaimohinh

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

type UseCase interface {
	List(ctx context.Context) ([]LoaiMoHinh, error)
}

type Handler struct {
	service UseCase
}

type loaiMoHinhResponse struct {
	IDLoaiMoHinh  string `json:"id_loai_mo_hinh"`
	TenLoaiMoHinh string `json:"ten_loai_mo_hinh"`
	MoTa          string `json:"mo_ta,omitempty"`
}

func NewHandler(service UseCase) (*Handler, error) {
	if service == nil {
		return nil, errors.New("model type service is required")
	}
	return &Handler{service: service}, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		httpx.WriteError(w, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"query parameters are not supported",
			nil,
		))
		return
	}

	items, err := h.service.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]loaiMoHinhResponse, len(items))
	for i, item := range items {
		response[i] = loaiMoHinhResponse{
			IDLoaiMoHinh:  item.IDLoaiMoHinh,
			TenLoaiMoHinh: item.TenLoaiMoHinh,
			MoTa:          item.MoTa,
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write list model types response: %v", err)
	}
}
