CREATE TABLE bao_cao_sos (
    id_bao_cao_sos UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_nguoi_dung UUID NOT NULL,
    id_toa_nha UUID NOT NULL,
    id_tang UUID NULL,
    id_khu_vuc UUID NULL,
    loai_sos VARCHAR(50) NULL,
    hinh_anh TEXT,
    mo_ta TEXT,
    trang_thai VARCHAR(50) DEFAULT 'NEW',
    ngay_tao TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_sos_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung),
    CONSTRAINT fk_sos_toa_nha FOREIGN KEY (id_toa_nha) REFERENCES toa_nha(id_toa_nha) ON DELETE CASCADE,
    CONSTRAINT fk_sos_tang FOREIGN KEY (id_tang) REFERENCES tang(id_tang) ON DELETE SET NULL,
    CONSTRAINT fk_sos_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE SET NULL
);

-- ==========================================
-- 26. LỊCH SỬ XỬ LÝ SOS
-- ==========================================
CREATE TABLE lich_su_xu_ly_sos (
    id_lich_su_xu_ly_sos UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_bao_cao_sos UUID NOT NULL,
    id_nguoi_dung UUID NOT NULL,
    hanh_dong VARCHAR(100) NOT NULL,
    trang_thai_cu VARCHAR(50),
    trang_thai_moi VARCHAR(50),
    ghi_chu TEXT,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_ls_sos_bao_cao FOREIGN KEY (id_bao_cao_sos) REFERENCES bao_cao_sos(id_bao_cao_sos) ON DELETE CASCADE,
    CONSTRAINT fk_ls_sos_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung)
);