
-- Bảng vai trò
CREATE TABLE vai_tro (
    id_vai_tro UUID PRIMARY KEY,
    ky_hieu_vai_tro VARCHAR(30) NOT NULL,
    ten_vai_tro VARCHAR(30) NOT NULL UNIQUE,
    mo_ta TEXT
);

-- Bảng người dùng
CREATE TABLE nguoi_dung (
    id_nguoi_dung UUID PRIMARY KEY,
    id_vai_tro UUID NOT NULL,

    ten_nguoi_dung VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    so_dien_thoai VARCHAR(15),
    mat_khau TEXT NOT NULL,

    ngay_tao TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    trang_thai VARCHAR(150) NOT NULL, --DangHoatDong / BiKhoa

    CONSTRAINT fk_nguoi_dung_vai_tro
        FOREIGN KEY (id_vai_tro)
        REFERENCES vai_tro (id_vai_tro)
        ON UPDATE CASCADE
        ON DELETE RESTRICT
);
-- Index sẽ đánh sau nha

-- -- Index hỗ trợ truy vấn người dùng theo vai trò
-- CREATE INDEX idx_nguoi_dung_id_vai_tro
--     ON nguoi_dung (id_vai_tro);

-- -- Index hỗ trợ lọc người dùng chưa bị xóa
-- CREATE INDEX idx_nguoi_dung_chua_xoa
--     ON nguoi_dung (id_nguoi_dung)
--     WHERE da_xoa = FALSE;
