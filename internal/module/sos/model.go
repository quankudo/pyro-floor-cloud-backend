package sos

import "time"

type SOSReport struct {
	IDBaoCaoSOS       string
	TenNguoiGui       string
	SDTNguoiGui       string
	LoaiSOS           string
	MoTa              string
	HinhAnh           string
	IDTang            *string
	SoTang            *int
	IDKhuVuc          *string
	TenKhuVuc         string
	ThoiGianBatDau    time.Time
	ThoiGianKetThuc   *time.Time
	ThoiGianTiepNhan  *time.Time
	ThoiLuongXuLyGiay *int64
	TrangThai         string
	IDNguoiPhuTrach   *string
	TenNguoiPhuTrach  string
	SDTNguoiPhuTrach  string
	GhiChuXuLy        string
}

type ListFilter struct {
	LoaiSOS  *string
	FromTime *time.Time
	ToTime   *time.Time
	Keyword  *string
	Limit    int
	Offset   int64
}

type ActiveFilter struct {
	LoaiSOS          *string
	TrangThai        *string
	ChiThuocPhanCong bool
	Limit            int
	Offset           int64
}

type ActiveSOSReport struct {
	IDBaoCaoSOS         string
	TenNguoiGui         string
	SDTNguoiGui         string
	LoaiSOS             string
	TrangThai           string
	MoTa                string
	HinhAnh             string
	ThoiGianGui         time.Time
	IDTang              *string
	SoTang              *int
	IDKhuVuc            *string
	TenKhuVuc           string
	ThuocPhanCongCuaToi bool
}

type Page struct {
	Items  []SOSReport
	Total  int
	Limit  int
	Offset int64
}

type ActivePage struct {
	Items  []ActiveSOSReport
	Total  int
	Limit  int
	Offset int64
}
