package phanconggiamsat

import "time"

const (
	StatusActive       = "ACTIVE"
	StatusLegacyActive = "DA_PHAN_CONG"
)

type PhanCongGiamSat struct {
	ID           string
	IDNguoiDung  string
	IDToaNha     string
	IDTang       *string
	IDKhuVuc     *string
	LoaiPhanCong string
	TrangThai    string
	NgayTao      time.Time
	NgayCapNhat  time.Time
}

type AddAssignmentInput struct {
	UserID  string
	FloorID string
	AreaID  string
}

type AddAssignmentsInput struct {
	BuildingID  string
	Assignments []AddAssignmentInput
}

type AssignmentScopeFilter struct {
	UserID           *string
	FloorNumber      *int
	AreaName         *string
	AssignmentStatus *string
}

type AssignmentScope struct {
	IDTang            string
	SoTang            int
	IDKhuVuc          *string
	TenKhuVuc         *string
	IDNguoiDung       *string
	TenNguoiDung      *string
	TrangThaiPhanCong string
}
