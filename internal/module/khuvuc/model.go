package khuvuc

type KhuVuc struct {
	IDKhuVuc     string
	IDTang       string
	IDLoaiKhuVuc string
	TenKhuVuc    string
	MoTa         string
}

type MonitoringFilter struct {
	FloorNumber       *int
	FloorOffset       int
	SearchKeyword     *string
	FireAlertOnly     *bool
	HardwareErrorOnly *bool
}

type RoomMonitoring struct {
	SoTang            int
	IDTang            string
	IDKhuVuc          string
	TenKhuVuc         string
	MoTaKhuVuc        string
	TenLoaiKhuVuc     string
	MucDoCanhBao      string
	TrangThaiCanhBao  string
	MoTaCanhBao       string
	CoCanhBao         bool
	CoLoiPhanCung     bool
	TrangThaiPhanCung string
	NhietDo           *float64
	DonViNhietDo      string
	NongDoKhoi        *float64
	DonViKhoi         string
	MoTaTinhTrang     string
}

type FloorMonitoring struct {
	IDTang        string
	SoTang        int
	SoLuongKhuVuc int
	MauTrangThai  string
	MaTrangThai   string
}

type BuildingAreaFilter struct {
	AreaTypeID    *string
	FloorNumber   *int
	StatusCode    *string
	SearchKeyword *string
	Limit         int
	Offset        int64
}

type BuildingArea struct {
	IDKhuVuc      string
	TenKhuVuc     string
	MoTaKhuVuc    string
	IDTang        string
	SoTang        int
	IDLoaiKhuVuc  string
	TenLoaiKhuVuc string
	SoLuongCuDan  int
	MauTrangThai  string
	MaTrangThai   string
}

type BuildingFloor struct {
	IDTang string
	SoTang int
}

type BuildingAreaType struct {
	IDLoaiKhuVuc  string
	TenLoaiKhuVuc string
}

type BuildingAreaPage struct {
	Items      []BuildingArea
	TotalCount int
	Floors     []BuildingFloor
	AreaTypes  []BuildingAreaType
	Limit      int
	Offset     int64
}

type KhuVucShort struct {
	IDKhuVuc      string
	TenKhuVuc     string
	IDLoaiKhuVuc  string
	TenLoaiKhuVuc string
}
