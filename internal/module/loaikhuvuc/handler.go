package loaikhuvuc

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

type UseCase interface {
	ListByModelTypeID(ctx context.Context, modelTypeID string) ([]LoaiKhuVuc, error)
}

type Handler struct {
	service UseCase
}

type loaiKhuVucResponse struct {
	IDLoaiKhuVuc         string `json:"id_loai_khu_vuc"`
	IDLoaiMoHinh         string `json:"id_loai_mo_hinh"`
	TenLoaiKhuVuc        string `json:"ten_loai_khu_vuc"`
	ThoiGianChoXacNhan   int    `json:"thoi_gian_cho_xac_nhan"`
	MucDoFallbackMacDinh string `json:"muc_do_fallback_mac_dinh"`
	SuDungGiaTriCamBien  bool   `json:"su_dung_gia_tri_cam_bien"`
	MoTa                 string `json:"mo_ta,omitempty"`
}

func NewHandler(service UseCase) (*Handler, error) {
	if service == nil {
		return nil, errors.New("area type service is required")
	}
	return &Handler{service: service}, nil
}

func (h *Handler) ListByModelTypeID(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		httpx.WriteError(w, apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"query parameters are not supported",
			nil,
		))
		return
	}

	items, err := h.service.ListByModelTypeID(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	response := make([]loaiKhuVucResponse, len(items))
	for i, item := range items {
		response[i] = loaiKhuVucResponse{
			IDLoaiKhuVuc:         item.IDLoaiKhuVuc,
			IDLoaiMoHinh:         item.IDLoaiMoHinh,
			TenLoaiKhuVuc:        item.TenLoaiKhuVuc,
			ThoiGianChoXacNhan:   item.ThoiGianChoXacNhan,
			MucDoFallbackMacDinh: item.MucDoFallbackMacDinh,
			SuDungGiaTriCamBien:  item.SuDungGiaTriCamBien,
			MoTa:                 item.MoTa,
		}
	}
	if err := httpx.WriteSuccess(w, http.StatusOK, response); err != nil {
		log.Printf("write list area types response: %v", err)
	}
}
