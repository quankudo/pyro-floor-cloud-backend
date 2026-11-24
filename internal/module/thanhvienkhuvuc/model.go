package thanhvienkhuvuc

const StatusActive = "ACTIVE"

type ThanhVienKhuVuc struct {
	IDNguoiDung string
	IDKhuVuc    string
	TrangThai   string
	MoTa        string
}

type AddMemberInput struct {
	UserID string
	AreaID string
}

type AddMembersInput struct {
	BuildingID string
	Members    []AddMemberInput
}
