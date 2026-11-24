CREATE TABLE nhat_ky_xu_ly_canh_bao (
    id_nhat_ky_xu_ly_canh_bao UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_nguoi_thuc_hien UUID NULL,
    id_canh_bao UUID NOT NULL,
    nguon VARCHAR(50) NOT NULL,
    loai_hanh_dong VARCHAR(100) NOT NULL,
    ghi_chu TEXT,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_nk_nguoi_dung FOREIGN KEY (id_nguoi_thuc_hien) REFERENCES nguoi_dung(id_nguoi_dung) ON DELETE SET NULL,
    CONSTRAINT fk_nk_canh_bao FOREIGN KEY (id_canh_bao) REFERENCES canh_bao(id_canh_bao) ON DELETE CASCADE
);

CREATE TABLE su_kien_canh_bao (
    id_su_kien_canh_bao UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_canh_bao UUID NOT NULL,
    id_cam_bien UUID NOT NULL,
    loai_su_kien VARCHAR(50) NOT NULL,
    gia_tri NUMERIC(10,2) NOT NULL,
    nguong NUMERIC(10,2) NOT NULL,
    muc_do VARCHAR(50) NOT NULL,
    thoi_gian_phat_hien TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_sk_canh_bao FOREIGN KEY (id_canh_bao) REFERENCES canh_bao(id_canh_bao) ON DELETE CASCADE,
    CONSTRAINT fk_sk_cam_bien FOREIGN KEY (id_cam_bien) REFERENCES cam_bien(id_cam_bien) ON DELETE CASCADE
);