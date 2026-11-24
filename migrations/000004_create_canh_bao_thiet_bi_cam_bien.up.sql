CREATE TABLE canh_bao (
    id_canh_bao UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_khu_vuc UUID NOT NULL,
    muc_do VARCHAR(50) NOT NULL,
    trang_thai VARCHAR(50) DEFAULT 'NEW',
    thoi_gian_phat_hien TIMESTAMP DEFAULT NOW(),
    thoi_gian_xac_nhan TIMESTAMP NULL,
    thoi_gian_tiep_nhan TIMESTAMP NULL,
    thoi_gian_ket_thuc TIMESTAMP NULL,
    mo_ta TEXT,
    CONSTRAINT fk_canh_bao_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE CASCADE
);

CREATE TABLE thiet_bi (
    id_thiet_bi UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_khu_vuc UUID NOT NULL,
    ma_thiet_bi VARCHAR(100) UNIQUE NOT NULL,
    ip_address VARCHAR(20) UNIQUE NOT NULL,
    phien_ban_firmware VARCHAR(50) NOT NULL,
    loai_thiet_bi VARCHAR(50) NOT NULL,
    trang_thai VARCHAR(50) DEFAULT 'ONLINE',
    lan_cuoi_hoat_dong TIMESTAMP NULL,
    CONSTRAINT fk_thiet_bi_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE CASCADE
);

CREATE TABLE cam_bien (
    id_cam_bien UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_thiet_bi UUID NOT NULL,
    ma_cam_bien VARCHAR(100) UNIQUE NOT NULL,
    loai_cam_bien VARCHAR(50) NOT NULL,
    gia_tri_hien_tai NUMERIC(10,2),
    don_vi_do VARCHAR(20),
    da_ghi_de BOOLEAN NULL,
    cau_hinh_nguong_json JSONB,
    trang_thai VARCHAR(20) DEFAULT 'ACTIVE',
    nha_san_xuat VARCHAR(100),
    vi_tri_lap_dat VARCHAR(150),
    thoi_gian_lap_dat TIMESTAMP DEFAULT NOW(),
    mo_ta TEXT,
    CONSTRAINT fk_cam_bien_thiet_bi FOREIGN KEY (id_thiet_bi) REFERENCES thiet_bi(id_thiet_bi) ON DELETE CASCADE
);