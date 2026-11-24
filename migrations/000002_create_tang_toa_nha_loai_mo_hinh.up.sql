CREATE TABLE loai_mo_hinh (
    id_loai_mo_hinh UUID PRIMARY KEY,
    ten_loai_mo_hinh VARCHAR(100) NOT NULL,
    mo_ta TEXT
);

CREATE TABLE toa_nha (
    id_toa_nha UUID PRIMARY KEY,
    id_nguoi_chu UUID NOT NULL,
    id_loai_mo_hinh UUID NOT NULL,
    so_tang INT NOT NULL,
    ten_toa_nha VARCHAR(150) NOT NULL,
    dia_chi TEXT,
    trang_thai VARCHAR(20) DEFAULT 'HoatDong',
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    mo_ta TEXT,
    CONSTRAINT fk_toa_nha_nguoi_chu FOREIGN KEY (id_nguoi_chu) REFERENCES nguoi_dung(id_nguoi_dung),
    CONSTRAINT fk_toa_nha_mo_hinh FOREIGN KEY (id_loai_mo_hinh) REFERENCES loai_mo_hinh(id_loai_mo_hinh)
);

CREATE TABLE tang (
    id_tang UUID PRIMARY KEY,
    id_toa_nha UUID NOT NULL,
    so_tang INT NOT NULL,
    CONSTRAINT fk_tang_toa_nha FOREIGN KEY (id_toa_nha) REFERENCES toa_nha(id_toa_nha) ON DELETE CASCADE
);

