package toanha

import (
	"time"
)

type ToaNha struct {
	IDToaNha     string
	IDNguoiChu   string
	IDLoaiMoHinh string
	SoTang       int
	TenToaNha    string
	NgayTao      time.Time
	DiaChi       string
	NgayCapNhat  time.Time
	MoTa         string
	TrangThai    string
}

type ListFilter struct {
	TrangThai    string
	IDLoaiMoHinh string
	TenToaNha    string
	Limit        int
	Offset       int64
}

type Page struct {
	Items  []ToaNha
	Total  int64
	Limit  int
	Offset int64
}
