package canhbao

import (
	"encoding/json"
	"time"
)

type CanhBao struct {
	IDCanhBao        string
	IDKhuVuc         string
	MucDo            string
	TrangThai        string
	ThoiGianPhatHien time.Time
	ThoiGianXacNhan  *time.Time
	ThoiGianTiepNhan *time.Time
	ThoiGianKetThuc  *time.Time
	MoTa             string
}

type AlertListItem struct {
	IDCanhBao        string
	IDKhuVuc         string
	MucDo            string
	TrangThai        string
	ThoiGianPhatHien time.Time
	ThoiGianXacNhan  *time.Time
	ThoiGianTiepNhan *time.Time
	ThoiGianKetThuc  *time.Time
	MoTa             string
	IDTang           string
	SoTang           int
	TenKhuVuc        string
}

type ListFilter struct {
	MucDo        *string
	TrangThai    *string
	IDTang       *string
	IDKhuVuc     *string
	FromPhatHien *time.Time
	ToPhatHien   *time.Time
	Limit        int
	Offset       int64
}

type ActiveFilter struct {
	TrangThai        *string
	ChiThuocPhanCong bool
	Limit            int
	Offset           int64
}

type ActiveAlertListItem struct {
	IDCanhBao               string
	MucDo                   string
	TrangThai               string
	MoTa                    string
	ThoiGianPhatHien        time.Time
	ThoiGianXacNhan         *time.Time
	ThoiGianTiepNhan        *time.Time
	IDTang                  string
	SoTang                  int
	IDKhuVuc                string
	TenKhuVuc               string
	ThuocPhanCongCuaToi     bool
	IDNguoiTiepNhan         *string
	TenNguoiTiepNhan        string
	SDTNguoiTiepNhan        string
	ThoiGianThaoTacTiepNhan *time.Time
	DanhSachSuKien          json.RawMessage
}

type ActivePage struct {
	Items  []ActiveAlertListItem
	Total  int
	Limit  int
	Offset int64
}

type Page struct {
	Items  []AlertListItem
	Total  int
	Limit  int
	Offset int64
}
