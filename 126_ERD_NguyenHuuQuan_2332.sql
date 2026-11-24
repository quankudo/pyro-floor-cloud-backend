-- ============================================================
--  PyroFloor Database Schema  (Page-1)
--  Generated from ERD_Pyro_Floor_drawio.xml
-- ============================================================

-- Bật extension uuid (PostgreSQL)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==========================================
-- 1. LOẠI MÔ HÌNH
-- ==========================================
CREATE TABLE loai_mo_hinh (
    id_loai_mo_hinh UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ten_loai_mo_hinh VARCHAR(100) NOT NULL,
    mo_ta TEXT
);

-- ==========================================
-- 2. VAI TRÒ
-- ==========================================
CREATE TABLE vai_tro (
    id_vai_tro UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ten_vai_tro VARCHAR(50) NOT NULL,
    mo_ta TEXT
);

-- ==========================================
-- 3. LOẠI KHU VỰC
-- ==========================================
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

-- ==========================================
-- 4. NGƯỜI DÙNG
-- ==========================================
CREATE TABLE nguoi_dung (
    id_nguoi_dung UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_vai_tro UUID NOT NULL,
    ten_nguoi_dung VARCHAR(100) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    so_dien_thoai VARCHAR(14),
    mat_khau VARCHAR(255) NOT NULL,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    trang_thai VARCHAR(20) NOT NULL,
    CONSTRAINT fk_nguoi_dung_vai_tro FOREIGN KEY (id_vai_tro) REFERENCES vai_tro(id_vai_tro)
);


-- ==========================================
-- 5. TÒA NHÀ
-- ==========================================
CREATE TABLE toa_nha (
    id_toa_nha UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_nguoi_chu UUID NOT NULL,
    id_loai_mo_hinh UUID NOT NULL,
    so_tang INT NOT NULL,
    ten_toa_nha VARCHAR(150) NOT NULL,
    dia_chi TEXT,
    trang_thai VARCHAR(20) DEFAULT 'ACTIVE',
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    mo_ta TEXT,
    CONSTRAINT fk_toa_nha_nguoi_chu FOREIGN KEY (id_nguoi_chu) REFERENCES nguoi_dung(id_nguoi_dung),
    CONSTRAINT fk_toa_nha_mo_hinh FOREIGN KEY (id_loai_mo_hinh) REFERENCES loai_mo_hinh(id_loai_mo_hinh)
);

-- ==========================================
-- 6. MÁY CHỦ CỤC BỘ
-- ==========================================
CREATE TABLE may_chu_cuc_bo (
    id_may_chu_cuc_bo UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_toa_nha UUID NOT NULL,
    ma_may_chu VARCHAR(100) UNIQUE NOT NULL,
    ten_may_chu VARCHAR(150) NOT NULL,
    trang_thai VARCHAR(50) DEFAULT 'ONLINE',
    token_xac_thuc VARCHAR(255) NOT NULL,
    phien_ban_phan_mem VARCHAR(50),
    cau_hinh_json JSONB,
    lan_cuoi_ket_noi TIMESTAMP NULL,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_may_chu_toa_nha FOREIGN KEY (id_toa_nha) REFERENCES toa_nha(id_toa_nha) ON DELETE CASCADE
);

-- ==========================================
-- 7. TẦNG
-- ==========================================
CREATE TABLE tang (
    id_tang UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_toa_nha UUID NOT NULL,
    so_tang INT NOT NULL,
    CONSTRAINT fk_tang_toa_nha FOREIGN KEY (id_toa_nha) REFERENCES toa_nha(id_toa_nha) ON DELETE CASCADE
);

-- ==========================================
-- 8. KHU VỰC
-- ==========================================
CREATE TABLE khu_vuc (
    id_khu_vuc UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_tang UUID NOT NULL,
    id_loai_khu_vuc UUID NOT NULL,
    ten_khu_vuc VARCHAR(150) NOT NULL,
    mo_ta TEXT,
    CONSTRAINT fk_khu_vuc_tang FOREIGN KEY (id_tang) REFERENCES tang(id_tang) ON DELETE CASCADE,
    CONSTRAINT fk_khu_vuc_loai FOREIGN KEY (id_loai_khu_vuc) REFERENCES loai_khu_vuc(id_loai_khu_vuc)
);


-- ==========================================
-- 9. THIẾT BỊ NGƯỜI DÙNG
-- ==========================================
CREATE TABLE thiet_bi_nguoi_dung (
    id_thiet_bi UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_nguoi_dung UUID NOT NULL,
    ma_thiet_bi VARCHAR(150) UNIQUE NOT NULL,
    push_token TEXT,
    nen_tang VARCHAR(50),
    ten_thiet_bi VARCHAR(50),
    trang_thai_hoat_dong BOOLEAN DEFAULT TRUE,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    lan_cuoi_dang_nhap TIMESTAMP NULL,
    CONSTRAINT fk_tb_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung) ON DELETE CASCADE
);

-- ==========================================
-- 10. PHÂN CÔNG GIÁM SÁT
-- ==========================================
CREATE TABLE phan_cong_giam_sat (
    id_phan_cong_giam_sat UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_nguoi_dung UUID NOT NULL,
    id_toa_nha UUID NOT NULL,
    id_tang UUID NULL,
    id_khu_vuc UUID NULL,
    loai_phan_cong VARCHAR(30) NOT NULL DEFAULT 'KHU_VUC',
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_pc_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung) ON DELETE CASCADE,
    CONSTRAINT fk_pc_toa_nha FOREIGN KEY (id_toa_nha) REFERENCES toa_nha(id_toa_nha) ON DELETE CASCADE,
    CONSTRAINT fk_pc_tang FOREIGN KEY (id_tang) REFERENCES tang(id_tang) ON DELETE SET NULL,
    CONSTRAINT fk_pc_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE SET NULL
);

-- ==========================================
-- 11. THÀNH VIÊN KHU VỰC
-- ==========================================
CREATE TABLE thanh_vien_khu_vuc (
    id_nguoi_dung UUID NOT NULL,
    id_khu_vuc UUID NOT NULL,
    PRIMARY KEY (id_nguoi_dung, id_khu_vuc),
    CONSTRAINT fk_tv_kv_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung) ON DELETE CASCADE,
    CONSTRAINT fk_tv_kv_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE CASCADE
);


-- ==========================================
-- 12. THIẾT BỊ
-- ==========================================
CREATE TABLE thiet_bi (
    id_thiet_bi UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_khu_vuc UUID NOT NULL,
    ma_thiet_bi VARCHAR(100) UNIQUE NOT NULL,
    ip_address VARCHAR(20) UNIQUE NOT NULL,
    phien_ban_firmware VARCHAR(50) UNIQUE NOT NULL,
    loai_thiet_bi VARCHAR(50) NOT NULL,
    trang_thai VARCHAR(50) DEFAULT 'ONLINE',
    lan_cuoi_hoat_dong TIMESTAMP NULL,
    CONSTRAINT fk_thiet_bi_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE CASCADE
);

-- ==========================================
-- 13. THIẾT BỊ CHẤP HÀNH
-- ==========================================
CREATE TABLE thiet_bi_chap_hanh (
    id_thiet_bi_chap_hanh UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_thiet_bi UUID NOT NULL,
    ma_thiet_bi VARCHAR(100) NOT NULL,
    loai_thiet_bi VARCHAR(50) NOT NULL,
    trang_thai VARCHAR(50) DEFAULT 'OFF',
    lan_cuoi_hoat_dong TIMESTAMP NULL,
    CONSTRAINT fk_tb_chap_hanh_tb FOREIGN KEY (id_thiet_bi) REFERENCES thiet_bi(id_thiet_bi) ON DELETE CASCADE
);

-- ==========================================
-- 14. CẢM BIẾN
-- ==========================================
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
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    mo_ta TEXT,
    CONSTRAINT fk_cam_bien_thiet_bi FOREIGN KEY (id_thiet_bi) REFERENCES thiet_bi(id_thiet_bi) ON DELETE CASCADE
);

-- ==========================================
-- 15. DỮ LIỆU CẢM BIẾN
-- ==========================================
CREATE TABLE du_lieu_cam_bien (
    id_du_lieu_cam_bien UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_cam_bien UUID NOT NULL,
    gia_tri NUMERIC(10,2) NOT NULL,
    thoi_gian_ghi_nhan TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_dl_cam_bien FOREIGN KEY (id_cam_bien) REFERENCES cam_bien(id_cam_bien) ON DELETE CASCADE
);

-- ==========================================
-- 16. CAMERA
-- ==========================================
CREATE TABLE camera (
    id_camera UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_khu_vuc UUID NOT NULL,
    ma_camera VARCHAR(100) UNIQUE NOT NULL,
    ten_camera VARCHAR(100) UNIQUE NOT NULL,
    rtsp_url TEXT NOT NULL,
    nha_san_xuat VARCHAR(100),
    vi_tri_lap_dat VARCHAR(150),
    trang_thai VARCHAR(50) DEFAULT 'ONLINE',
    CONSTRAINT fk_camera_khu_vuc FOREIGN KEY (id_khu_vuc) REFERENCES khu_vuc(id_khu_vuc) ON DELETE CASCADE
);

-- ==========================================
-- 17. VỊ TRÍ PRESET CAMERA
-- ==========================================
CREATE TABLE vi_tri_preset_camera (
    id_preset UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_camera UUID NOT NULL,
    id_cam_bien UUID NULL,
    ma_preset VARCHAR(50) NOT NULL,
    ten_preset VARCHAR(50),
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_preset_camera FOREIGN KEY (id_camera) REFERENCES camera(id_camera) ON DELETE CASCADE,
    CONSTRAINT fk_preset_cam_bien FOREIGN KEY (id_cam_bien) REFERENCES cam_bien(id_cam_bien) ON DELETE SET NULL
);

-- ==========================================
-- 18. CẤU HÌNH NGƯỠNG CẢM BIẾN
-- ==========================================
CREATE TABLE cau_hinh_nguong_cam_bien (
    id_cau_hinh_nguong_cam_bien UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_loai_khu_vuc UUID NOT NULL,
    loai_cam_bien VARCHAR(50) NOT NULL,
    cau_hinh_nguong_json JSONB NOT NULL,
    don_vi_do VARCHAR(20),
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_ch_nguong_loai_kv FOREIGN KEY (id_loai_khu_vuc) REFERENCES loai_khu_vuc(id_loai_khu_vuc) ON DELETE CASCADE
);

-- ==========================================
-- 19. KỊCH BẢN LIÊN ĐỘNG
-- ==========================================
CREATE TABLE kich_ban_lien_dong (
    id_kich_ban_lien_dong UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_loai_khu_vuc UUID NOT NULL,
    ten_kich_ban VARCHAR(150) NOT NULL,
    muc_do_kich_hoat VARCHAR(50) NOT NULL,
    hanh_dong_chap_hanh_json JSONB NOT NULL,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    ngay_cap_nhat TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_kb_loai_khu_vuc FOREIGN KEY (id_loai_khu_vuc) REFERENCES loai_khu_vuc(id_loai_khu_vuc) ON DELETE CASCADE,
    CONSTRAINT uk_loai_khu_vuc_muc_do UNIQUE (id_loai_khu_vuc, muc_do_kich_hoat)
);


-- ==========================================
-- 20. CẢNH BÁO
-- ==========================================
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

-- ==========================================
-- 21. SỰ KIỆN CẢNH BÁO
-- ==========================================
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

-- ==========================================
-- 22. ẢNH XÁC MINH
-- ==========================================
CREATE TABLE anh_xac_minh (
    id_anh_xac_minh UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_canh_bao UUID NOT NULL,
    id_camera UUID NOT NULL,
    duong_dan TEXT NOT NULL,
    ket_qua_du_doan TEXT,
    thoi_gian_phat_hien TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_anh_canh_bao FOREIGN KEY (id_canh_bao) REFERENCES canh_bao(id_canh_bao) ON DELETE CASCADE,
    CONSTRAINT fk_anh_camera FOREIGN KEY (id_camera) REFERENCES camera(id_camera) ON DELETE CASCADE
);

-- ==========================================
-- 23. NHẬT KÝ XỬ LÝ CẢNH BÁO
-- ==========================================
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

-- ==========================================
-- 24. THÔNG BÁO
-- ==========================================
CREATE TABLE thong_bao (
    id_thong_bao UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_canh_bao UUID NULL,
    id_nguoi_dung UUID NOT NULL,
    tieu_de VARCHAR(150) NOT NULL,
    loai_thong_bao VARCHAR(50) NOT NULL,
    noi_dung TEXT NOT NULL,
    da_doc BOOLEAN DEFAULT FALSE,
    ngay_tao TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_tb_canh_bao FOREIGN KEY (id_canh_bao) REFERENCES canh_bao(id_canh_bao) ON DELETE CASCADE,
    CONSTRAINT fk_tb_nguoi_dung FOREIGN KEY (id_nguoi_dung) REFERENCES nguoi_dung(id_nguoi_dung) ON DELETE CASCADE
);

-- ==========================================
-- 25. BÁO CÁO SOS
-- ==========================================
CREATE TABLE bao_cao_sos (
    id_bao_cao_sos UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_nguoi_dung UUID NOT NULL,
    id_toa_nha UUID NOT NULL,
    id_tang UUID NULL,
    id_khu_vuc UUID NULL,
    loai_sos NULL,
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

-- ============================================================
-- ESSENTIAL INDEXES - TỐI ƯU CHO HIỆU NĂNG GHI & TRUY VẤN NÓNG
-- ============================================================

CREATE INDEX idx_nguoi_dung_email ON nguoi_dung(email);
CREATE INDEX idx_du_lieu_cam_bien_sensor_time ON du_lieu_cam_bien(id_cam_bien, thoi_gian_ghi_nhan DESC);
CREATE INDEX idx_canh_bao_trang_thai_thoi_gian ON canh_bao(trang_thai, thoi_gian_phat_hien DESC);
CREATE INDEX idx_bao_cao_sos_trang_thai_ngay ON bao_cao_sos(trang_thai, ngay_tao DESC);
CREATE INDEX idx_thong_bao_chua_doc ON thong_bao(id_nguoi_dung) WHERE da_doc = FALSE;
CREATE INDEX idx_su_kien_cb_canh_bao ON su_kien_canh_bao(id_canh_bao);
CREATE INDEX idx_nhat_ky_xu_ly_cb ON nhat_ky_xu_ly_canh_bao(id_canh_bao);