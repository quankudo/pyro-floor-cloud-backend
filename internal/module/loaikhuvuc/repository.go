package loaikhuvuc

import "context"

type Repository interface {
	ListByModelTypeID(ctx context.Context, modelTypeID string) ([]LoaiKhuVuc, error)
}
