package nguoidung

type NguoiDung struct {
	IDNguoiDung  string
	IDVaiTro     string
	TenNguoiDung string
	Email        string
	SoDienThoai  string
	NgayTao      string
	NgayCapNhat  string
	TrangThai    TrangThai
}

type TrangThai string

const HoatDong TrangThai = "DangHoatDong"
const DaXoa TrangThai = "DaXoa"

const (
	RoleCodeAdmin    = "QUAN_TRI_VIEN"
	RoleCodeOwner    = "CHU_SO_HUU"
	RoleCodeManager  = "QUAN_LY"
	RoleCodeGuard    = "BAO_VE"
	RoleCodeResident = "CU_DAN"
)

type VaiTro struct {
	IDVaiTro     string
	KyHieuVaiTro string
}

type ManagerAssignment struct {
	IDTang   string
	IDKhuVuc string
}

type CreateUserInput struct {
	TenNguoiDung string
	Email        string
	SoDienThoai  string
	MatKhauHash  string
	VaiTro       VaiTro
	IDKhuVuc     string
	PhanCong     *ManagerAssignment
}

type CreatedUser struct {
	NguoiDung
	KyHieuVaiTro string
}

type LoginRecord struct {
	User         NguoiDung
	PasswordHash string
	Role         string
	IDToaNha     string
	IDKhuVuc     string
}

type ResidentFilter struct {
	FloorID       *string
	AreaID        *string
	Status        *string
	SearchKeyword *string
	Limit         int
	Offset        int64
	IncludeFloors bool
}

type Resident struct {
	IDNguoiDung  string
	TenNguoiDung string
	Email        string
	SoDienThoai  string
	TrangThai    string
	NgayTao      string
	IDKhuVuc     string
	TenKhuVuc    string
	IDTang       string
	SoTang       int
}

type ResidentPage struct {
	Items  []Resident
	Floors []ResidentFloor
	Total  int
	Limit  int
	Offset int64
}

type ResidentFloor struct {
	IDTang string
	SoTang int
}

type SupervisorActorFilter struct {
	SearchKeyword *string
	Status        *string
	Limit         int
	Offset        int64
}

type SupervisorActor struct {
	IDNguoiDung       string
	TenNguoiDung      string
	Email             string
	SoDienThoai       string
	TrangThai         string
	NgayTao           string
	IDVaiTro          string
	KyHieuVaiTro      string
	TenVaiTro         string
	TrangThaiPhanCong string
}

type SupervisorActorPage struct {
	Items  []SupervisorActor
	Total  int
	Limit  int
	Offset int64
}
