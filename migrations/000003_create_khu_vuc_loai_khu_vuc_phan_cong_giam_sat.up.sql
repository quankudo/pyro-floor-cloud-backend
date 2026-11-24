CREATE TABLE loai_khu_vuc (
    id_loai_khu_vuc UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_loai_mo_hinh UUID NOT NULL,
    ten_loai_khu_vuc VARCHAR(100) NOT NULL,
    thoi_gian_cho_xac_nhan INT DEFAULT 30,
    muc_do_fallback_mac_dinh VARCHAR(50) DEFAULT 'DANGER',
    su_dung_gia_tri_cam_bien BOOLEAN DEFAULT TRUE,
    mo_ta TEXT,
    CONSTRAINT fk_loai_kv_mo_hinh FOREIGN KEY (id_loai_mo_hinh) REFERENCES loai_mo_hinh(id_loai_mo_hinh)
);

CREATE TABLE khu_vuc (
    id_khu_vuc UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_tang UUID NOT NULL,
    id_loai_khu_vuc UUID NOT NULL,
    ten_khu_vuc VARCHAR(150) NOT NULL,
    mo_ta TEXT,
    CONSTRAINT fk_khu_vuc_tang FOREIGN KEY (id_tang) REFERENCES tang(id_tang) ON DELETE CASCADE,
    CONSTRAINT fk_khu_vuc_loai FOREIGN KEY (id_loai_khu_vuc) REFERENCES loai_khu_vuc(id_loai_khu_vuc)
);

CREATE TABLE thanh_vien_khu_vuc (
    id_nguoi_dung UUID NOT NULL,
    id_khu_vuc UUID NOT NULL,
    trang_thai VARCHAR(50) NOT NULL,
    mo_ta TEXT,
    PRIMARY KEY (id_nguoi_dung, id_khu_vuc),
    CONSTRAINT fk_tv_kv_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung) ON DELETE CASCADE,
    CONSTRAINT fk_tv_kv_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE CASCADE
);

CREATE TABLE phan_cong_giam_sat (
    id_phan_cong_giam_sat UUID PRIMARY KEY,
    id_nguoi_dung UUID NOT NULL,
    id_toa_nha UUID NOT NULL,
    id_tang UUID NULL,
    id_khu_vuc UUID NULL,
    loai_phan_cong VARCHAR(30) NOT NULL, -- TOA_NHA, TANG, KHU_VUC
    trang_thai VARCHAR(50) NOT NULL,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_pc_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung) ON DELETE CASCADE,
    CONSTRAINT fk_pc_toa_nha FOREIGN KEY (id_toa_nha) REFERENCES toa_nha(id_toa_nha) ON DELETE CASCADE,
    CONSTRAINT fk_pc_tang FOREIGN KEY (id_tang) REFERENCES tang(id_tang) ON DELETE SET NULL,
    CONSTRAINT fk_pc_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE SET NULL
);