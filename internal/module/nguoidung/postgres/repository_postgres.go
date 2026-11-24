package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pyrofloor/cloud-backend/internal/database"
	"github.com/pyrofloor/cloud-backend/internal/module/nguoidung"
)

type NguoiDungRepository struct {
	db *pgxpool.Pool
}

func NewNguoiDungRepository(db *pgxpool.Pool) *NguoiDungRepository {
	return &NguoiDungRepository{
		db: db,
	}
}

func (r *NguoiDungRepository) ResolveRole(
	ctx context.Context,
	roleID, roleCode string,
) (nguoidung.VaiTro, error) {
	var (
		role nguoidung.VaiTro
		err  error
	)
	if roleID != "" {
		err = r.db.QueryRow(
			ctx,
			`SELECT id_vai_tro::text, ky_hieu_vai_tro
			FROM vai_tro
			WHERE id_vai_tro = $1::uuid`,
			roleID,
		).Scan(&role.IDVaiTro, &role.KyHieuVaiTro)
	} else {
		rows, queryErr := r.db.Query(
			ctx,
			`SELECT id_vai_tro::text, ky_hieu_vai_tro
			FROM vai_tro
			WHERE UPPER(ky_hieu_vai_tro) = $1
			ORDER BY id_vai_tro
			LIMIT 2`,
			roleCode,
		)
		if queryErr != nil {
			return nguoidung.VaiTro{}, fmt.Errorf("resolve role by code: %w", queryErr)
		}
		defer rows.Close()

		if !rows.Next() {
			if rowErr := rows.Err(); rowErr != nil {
				return nguoidung.VaiTro{}, fmt.Errorf("read role by code: %w", rowErr)
			}
			return nguoidung.VaiTro{}, nguoidung.ErrRoleNotFound
		}
		if err := rows.Scan(&role.IDVaiTro, &role.KyHieuVaiTro); err != nil {
			return nguoidung.VaiTro{}, fmt.Errorf("scan role by code: %w", err)
		}
		if rows.Next() {
			return nguoidung.VaiTro{}, errors.New("role code is not unique")
		}
		if err := rows.Err(); err != nil {
			return nguoidung.VaiTro{}, fmt.Errorf("iterate role rows: %w", err)
		}
		return role, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nguoidung.VaiTro{}, nguoidung.ErrRoleNotFound
	}
	if err != nil {
		return nguoidung.VaiTro{}, fmt.Errorf("resolve role by ID: %w", err)
	}
	return role, nil
}

func (r *NguoiDungRepository) WithinTransaction(
	ctx context.Context,
	fn func(database.DBTX) error,
) (returnErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin batch user transaction: %w", err)
	}
	defer func() {
		if returnErr == nil {
			return
		}
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil &&
			!errors.Is(rollbackErr, pgx.ErrTxClosed) {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("rollback batch user transaction: %w", rollbackErr),
			)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit batch user transaction: %w", err)
	}
	return nil
}

func (r *NguoiDungRepository) FindOrCreateBatchUser(
	ctx context.Context,
	tx database.DBTX,
	user nguoidung.CreateUserInput,
) (nguoidung.CreatedUser, error) {
	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended(LOWER($1), 0))`,
		user.Email,
	); err != nil {
		return nguoidung.CreatedUser{}, fmt.Errorf("lock user email for batch add: %w", err)
	}

	created, err := findBatchUserByEmail(ctx, tx, user.Email)
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nguoidung.CreatedUser{}, err
	}

	const insertQuery = `
		INSERT INTO nguoi_dung (
			id_nguoi_dung,
			id_vai_tro,
			ten_nguoi_dung,
			email,
			so_dien_thoai,
			mat_khau,
			trang_thai
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, NULLIF($5, ''), $6, $7)
		ON CONFLICT (email) DO NOTHING
		RETURNING ngay_tao, ngay_cap_nhat
	`

	userID := uuid.NewString()
	var createdAt, updatedAt time.Time
	err = tx.QueryRow(
		ctx,
		insertQuery,
		userID,
		user.VaiTro.IDVaiTro,
		user.TenNguoiDung,
		user.Email,
		user.SoDienThoai,
		user.MatKhauHash,
		nguoidung.HoatDong,
	).Scan(&createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, findErr := findBatchUserByEmail(ctx, tx, user.Email)
		if findErr != nil {
			return nguoidung.CreatedUser{}, fmt.Errorf("find user after email conflict: %w", findErr)
		}
		return existing, nil
	}
	if err != nil {
		return nguoidung.CreatedUser{}, fmt.Errorf("insert batch user: %w", err)
	}

	created = nguoidung.CreatedUser{
		NguoiDung: nguoidung.NguoiDung{
			IDNguoiDung:  userID,
			IDVaiTro:     user.VaiTro.IDVaiTro,
			TenNguoiDung: user.TenNguoiDung,
			Email:        user.Email,
			SoDienThoai:  user.SoDienThoai,
			NgayTao:      createdAt.UTC().Format(time.RFC3339Nano),
			NgayCapNhat:  updatedAt.UTC().Format(time.RFC3339Nano),
			TrangThai:    nguoidung.HoatDong,
		},
		KyHieuVaiTro: user.VaiTro.KyHieuVaiTro,
	}
	return created, nil
}

func findBatchUserByEmail(
	ctx context.Context,
	tx database.DBTX,
	email string,
) (nguoidung.CreatedUser, error) {
	const findQuery = `
		SELECT
			user_account.id_nguoi_dung::text,
			user_account.id_vai_tro::text,
			role.ky_hieu_vai_tro,
			user_account.ten_nguoi_dung,
			user_account.email,
			COALESCE(user_account.so_dien_thoai, ''),
			user_account.trang_thai,
			user_account.ngay_tao,
			user_account.ngay_cap_nhat
		FROM nguoi_dung AS user_account
		INNER JOIN vai_tro AS role
			ON role.id_vai_tro = user_account.id_vai_tro
		WHERE LOWER(user_account.email) = LOWER($1)
		ORDER BY user_account.id_nguoi_dung
		LIMIT 1
		FOR UPDATE OF user_account
	`
	var created nguoidung.CreatedUser
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx, findQuery, email).Scan(
		&created.IDNguoiDung,
		&created.IDVaiTro,
		&created.KyHieuVaiTro,
		&created.TenNguoiDung,
		&created.Email,
		&created.SoDienThoai,
		&created.TrangThai,
		&createdAt,
		&updatedAt,
	)
	if err == nil {
		created.NgayTao = createdAt.UTC().Format(time.RFC3339Nano)
		created.NgayCapNhat = updatedAt.UTC().Format(time.RFC3339Nano)
		return created, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nguoidung.CreatedUser{}, pgx.ErrNoRows
	}
	return nguoidung.CreatedUser{}, fmt.Errorf("find existing user by email: %w", err)
}

func (r *NguoiDungRepository) CreateUser(
	ctx context.Context,
	name, email string,
	phone string,
	passwordHash string,
) error {
	const query = `
		INSERT INTO nguoi_dung (
			id_nguoi_dung,
			id_vai_tro,
			ten_nguoi_dung,
			email,
			so_dien_thoai,
			mat_khau,
			trang_thai
		)
		SELECT
			$1,
			id_vai_tro,
			$2,
			$3,
			$4,
			$5,
			'DangHoatDong'
		FROM vai_tro
		WHERE ky_hieu_vai_tro = 'CU_DAN'
	`

	_, err := r.db.Exec(
		ctx,
		query,
		uuid.NewString(),
		name,
		email,
		phone,
		passwordHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			(pgErr.ConstraintName == "nguoi_dung_email_key" ||
				pgErr.ConstraintName == "uq_nguoi_dung_email") {
			return nguoidung.ErrEmailExists
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *NguoiDungRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*nguoidung.LoginRecord, error) {
	const query = `
		SELECT
			nd.id_nguoi_dung,
			vt.ky_hieu_vai_tro,
			nd.ten_nguoi_dung,
			nd.email,
			nd.so_dien_thoai,
			nd.mat_khau,
			nd.trang_thai,
			nd.ngay_tao
		FROM nguoi_dung AS nd
		INNER JOIN vai_tro AS vt
			ON vt.id_vai_tro = nd.id_vai_tro
		WHERE nd.email = $1
	`

	var rec nguoidung.LoginRecord
	var createdAt time.Time
	var phone pgtype.Text

	err := r.db.QueryRow(
		ctx,
		query,
		email,
	).Scan(
		&rec.User.IDNguoiDung,
		&rec.Role,
		&rec.User.TenNguoiDung,
		&rec.User.Email,
		&phone,
		&rec.PasswordHash,
		&rec.User.TrangThai,
		&createdAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nguoidung.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if phone.Valid {
		rec.User.SoDienThoai = phone.String
	}

	rec.User.NgayTao = createdAt.UTC().Format(time.RFC3339Nano)

	return &rec, nil
}

func (r *NguoiDungRepository) FindResidentByEmail(
	ctx context.Context,
	email string,
) (*nguoidung.LoginRecord, error) {
	const query = `
		SELECT
			nd.id_nguoi_dung::text,
			nd.email,
			nd.mat_khau,
			nd.ten_nguoi_dung,
			nd.trang_thai,
			vt.ky_hieu_vai_tro,
			tvkv.id_khu_vuc::text,
			t.id_toa_nha::text
		FROM nguoi_dung AS nd
		INNER JOIN vai_tro AS vt ON nd.id_vai_tro = vt.id_vai_tro
		INNER JOIN thanh_vien_khu_vuc AS tvkv ON nd.id_nguoi_dung = tvkv.id_nguoi_dung
		INNER JOIN khu_vuc AS kv ON tvkv.id_khu_vuc = kv.id_khu_vuc
		INNER JOIN tang AS t ON kv.id_tang = t.id_tang
		WHERE LOWER(nd.email) = LOWER($1)
		ORDER BY tvkv.id_khu_vuc
		LIMIT 1
	`

	var rec nguoidung.LoginRecord
	err := r.db.QueryRow(ctx, query, email).Scan(
		&rec.User.IDNguoiDung,
		&rec.User.Email,
		&rec.PasswordHash,
		&rec.User.TenNguoiDung,
		&rec.User.TrangThai,
		&rec.Role,
		&rec.IDKhuVuc,
		&rec.IDToaNha,
	)
	return scanActorLoginRecord(err, &rec)
}

func (r *NguoiDungRepository) FindManagerByEmail(
	ctx context.Context,
	email string,
) (*nguoidung.LoginRecord, error) {
	const query = `
		SELECT
			nd.id_nguoi_dung::text,
			nd.email,
			nd.mat_khau,
			nd.ten_nguoi_dung,
			nd.trang_thai,
			vt.ky_hieu_vai_tro,
			COALESCE(
				pcgs.id_toa_nha,
				t_via_tang.id_toa_nha,
				t_via_kv.id_toa_nha
			)::text AS id_toa_nha
		FROM nguoi_dung AS nd
		INNER JOIN vai_tro AS vt ON nd.id_vai_tro = vt.id_vai_tro
		LEFT JOIN phan_cong_giam_sat AS pcgs ON nd.id_nguoi_dung = pcgs.id_nguoi_dung
		LEFT JOIN tang AS t_via_tang ON pcgs.id_tang = t_via_tang.id_tang
		LEFT JOIN khu_vuc AS kv_via_pc ON pcgs.id_khu_vuc = kv_via_pc.id_khu_vuc
		left JOIN tang AS t_via_kv ON kv_via_pc.id_tang = t_via_kv.id_tang
		WHERE LOWER(nd.email) = LOWER($1)
		ORDER BY pcgs.id_phan_cong_giam_sat
		LIMIT 1
	`

	var rec nguoidung.LoginRecord
	err := r.db.QueryRow(ctx, query, email).Scan(
		&rec.User.IDNguoiDung,
		&rec.User.Email,
		&rec.PasswordHash,
		&rec.User.TenNguoiDung,
		&rec.User.TrangThai,
		&rec.Role,
		&rec.IDToaNha,
	)
	return scanActorLoginRecord(err, &rec)
}

func (r *NguoiDungRepository) FindOwnerByEmail(
	ctx context.Context,
	email string,
) (*nguoidung.LoginRecord, error) {
	const query = `
		SELECT
			nd.id_nguoi_dung::text,
			nd.email,
			nd.mat_khau,
			nd.ten_nguoi_dung,
			nd.trang_thai,
			vt.ky_hieu_vai_tro,
			tn.id_toa_nha::text
		FROM nguoi_dung AS nd
		INNER JOIN vai_tro AS vt ON nd.id_vai_tro = vt.id_vai_tro
		INNER JOIN toa_nha AS tn ON nd.id_nguoi_dung = tn.id_nguoi_chu
		WHERE LOWER(nd.email) = LOWER($1)
		ORDER BY tn.id_toa_nha
		LIMIT 1
	`

	var rec nguoidung.LoginRecord
	err := r.db.QueryRow(ctx, query, email).Scan(
		&rec.User.IDNguoiDung,
		&rec.User.Email,
		&rec.PasswordHash,
		&rec.User.TenNguoiDung,
		&rec.User.TrangThai,
		&rec.Role,
		&rec.IDToaNha,
	)
	return scanActorLoginRecord(err, &rec)
}

func scanActorLoginRecord(
	err error,
	rec *nguoidung.LoginRecord,
) (*nguoidung.LoginRecord, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nguoidung.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find actor user by email: %w", err)
	}
	return rec, nil
}

func (r *NguoiDungRepository) ListResidents(
	ctx context.Context,
	buildingID string,
	filter nguoidung.ResidentFilter,
) ([]nguoidung.Resident, int, error) {
	var floorID, areaID, status, searchKeyword any
	if filter.FloorID != nil {
		floorID = *filter.FloorID
	}
	if filter.AreaID != nil {
		areaID = *filter.AreaID
	}
	if filter.Status != nil {
		status = *filter.Status
	}
	if filter.SearchKeyword != nil {
		searchKeyword = *filter.SearchKeyword
	}

	const listQuery = `
		SELECT DISTINCT ON (nd.id_nguoi_dung)
			nd.id_nguoi_dung,
			nd.ten_nguoi_dung,
			nd.email,
			nd.so_dien_thoai,
			nd.trang_thai,
			nd.ngay_tao,
			kv.id_khu_vuc,
			kv.ten_khu_vuc,
			t.id_tang,
			t.so_tang
		FROM nguoi_dung nd
		JOIN vai_tro vt
		  ON vt.id_vai_tro = nd.id_vai_tro
		JOIN thanh_vien_khu_vuc tv
		  ON tv.id_nguoi_dung = nd.id_nguoi_dung
		JOIN khu_vuc kv
		  ON kv.id_khu_vuc = tv.id_khu_vuc
		JOIN tang t
		  ON t.id_tang = kv.id_tang
		WHERE t.id_toa_nha = $1
		  AND vt.ky_hieu_vai_tro = 'CU_DAN'
		  AND tv.trang_thai = 'DANG_HOAT_DONG'
		  AND ($2::UUID IS NULL OR t.id_tang = $2)
		  AND ($3::UUID IS NULL OR kv.id_khu_vuc = $3)
		  AND ($4::TEXT IS NULL OR nd.trang_thai = $4)
		  AND (
		      $5::TEXT IS NULL
		      OR nd.ten_nguoi_dung ILIKE '%' || $5 || '%'
		      OR nd.email ILIKE '%' || $5 || '%'
		      OR nd.so_dien_thoai ILIKE '%' || $5 || '%'
		  )
		ORDER BY
		    nd.id_nguoi_dung,
		    t.so_tang ASC,
		    kv.ten_khu_vuc ASC,
		    nd.ngay_tao DESC
		LIMIT $6 OFFSET $7
	`
	rows, err := r.db.Query(
		ctx,
		listQuery,
		buildingID,
		floorID,
		areaID,
		status,
		searchKeyword,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list building residents: %w", err)
	}
	defer rows.Close()

	items := make([]nguoidung.Resident, 0, filter.Limit)
	for rows.Next() {
		var item nguoidung.Resident
		var phone pgtype.Text
		var createdAt time.Time
		if err := rows.Scan(
			&item.IDNguoiDung,
			&item.TenNguoiDung,
			&item.Email,
			&phone,
			&item.TrangThai,
			&createdAt,
			&item.IDKhuVuc,
			&item.TenKhuVuc,
			&item.IDTang,
			&item.SoTang,
		); err != nil {
			return nil, 0, fmt.Errorf("scan building resident row: %w", err)
		}
		if phone.Valid {
			item.SoDienThoai = phone.String
		}
		item.NgayTao = createdAt.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate building resident rows: %w", err)
	}

	const countQuery = `
		SELECT COUNT(DISTINCT nd.id_nguoi_dung)::INT AS total_items
		FROM nguoi_dung nd
		JOIN vai_tro vt
		  ON vt.id_vai_tro = nd.id_vai_tro
		JOIN thanh_vien_khu_vuc tv
		  ON tv.id_nguoi_dung = nd.id_nguoi_dung
		JOIN khu_vuc kv
		  ON kv.id_khu_vuc = tv.id_khu_vuc
		JOIN tang t
		  ON t.id_tang = kv.id_tang
		WHERE t.id_toa_nha = $1
		  AND vt.ky_hieu_vai_tro = 'CU_DAN'
		  AND tv.trang_thai = 'DANG_HOAT_DONG'
		  AND ($2::UUID IS NULL OR t.id_tang = $2)
		  AND ($3::UUID IS NULL OR kv.id_khu_vuc = $3)
		  AND ($4::TEXT IS NULL OR nd.trang_thai = $4)
		  AND (
		      $5::TEXT IS NULL
		      OR nd.ten_nguoi_dung ILIKE '%' || $5 || '%'
		      OR nd.email ILIKE '%' || $5 || '%'
		      OR nd.so_dien_thoai ILIKE '%' || $5 || '%'
		  )
	`
	var total int
	if err := r.db.QueryRow(
		ctx,
		countQuery,
		buildingID,
		floorID,
		areaID,
		status,
		searchKeyword,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count building residents: %w", err)
	}
	return items, total, nil
}

func (r *NguoiDungRepository) ListSupervisorActors(
	ctx context.Context,
	buildingID string,
	filter nguoidung.SupervisorActorFilter,
) ([]nguoidung.SupervisorActor, int, error) {
	var searchKeyword, status any
	if filter.SearchKeyword != nil {
		searchKeyword = *filter.SearchKeyword
	}
	if filter.Status != nil {
		status = *filter.Status
	}

	const listQuery = `
		SELECT DISTINCT ON (nd.id_nguoi_dung)
		    nd.id_nguoi_dung,
		    nd.ten_nguoi_dung,
		    nd.email,
		    nd.so_dien_thoai,
		    nd.trang_thai,
		    nd.ngay_tao,
		    vt.id_vai_tro,
		    vt.ky_hieu_vai_tro,
		    vt.ten_vai_tro,
		    pcgs.trang_thai AS trang_thai_phan_cong
		FROM nguoi_dung nd
		JOIN vai_tro vt
		  ON vt.id_vai_tro = nd.id_vai_tro
		JOIN phan_cong_giam_sat pcgs
		  ON pcgs.id_nguoi_dung = nd.id_nguoi_dung
		WHERE pcgs.id_toa_nha = $1
		  AND vt.ky_hieu_vai_tro = 'QUAN_LY'
		  AND (
		      $2::TEXT IS NULL
		      OR nd.ten_nguoi_dung ILIKE '%' || $2 || '%'
		      OR nd.email ILIKE '%' || $2 || '%'
		      OR nd.so_dien_thoai ILIKE '%' || $2 || '%'
		  )
		  AND ($3::TEXT IS NULL OR pcgs.trang_thai = $3)
		ORDER BY
		    nd.id_nguoi_dung,
		    nd.ten_nguoi_dung ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(
		ctx,
		listQuery,
		buildingID,
		searchKeyword,
		status,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list supervisor actors: %w", err)
	}
	defer rows.Close()

	items := make([]nguoidung.SupervisorActor, 0, filter.Limit)
	for rows.Next() {
		var item nguoidung.SupervisorActor
		var phone pgtype.Text
		var roleName pgtype.Text
		var createdAt time.Time
		if err := rows.Scan(
			&item.IDNguoiDung,
			&item.TenNguoiDung,
			&item.Email,
			&phone,
			&item.TrangThai,
			&createdAt,
			&item.IDVaiTro,
			&item.KyHieuVaiTro,
			&roleName,
			&item.TrangThaiPhanCong,
		); err != nil {
			return nil, 0, fmt.Errorf("scan supervisor actor row: %w", err)
		}
		if phone.Valid {
			item.SoDienThoai = phone.String
		}
		if roleName.Valid {
			item.TenVaiTro = roleName.String
		}
		item.NgayTao = createdAt.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate supervisor actor rows: %w", err)
	}

	const countQuery = `
		SELECT COUNT(DISTINCT nd.id_nguoi_dung)::INT AS total_items
		FROM nguoi_dung nd
		JOIN vai_tro vt
		  ON vt.id_vai_tro = nd.id_vai_tro
		JOIN phan_cong_giam_sat pcgs
		  ON pcgs.id_nguoi_dung = nd.id_nguoi_dung
		WHERE pcgs.id_toa_nha = $1
		  AND vt.ky_hieu_vai_tro = 'QUAN_LY'
		  AND (
		      $2::TEXT IS NULL
		      OR nd.ten_nguoi_dung ILIKE '%' || $2 || '%'
		      OR nd.email ILIKE '%' || $2 || '%'
		      OR nd.so_dien_thoai ILIKE '%' || $2 || '%'
		  )
		  AND ($3::TEXT IS NULL OR pcgs.trang_thai = $3)
	`
	var total int
	if err := r.db.QueryRow(
		ctx,
		countQuery,
		buildingID,
		searchKeyword,
		status,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count supervisor actors: %w", err)
	}
	return items, total, nil
}

func nullableText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}
