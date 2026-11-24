package loaimohinh

import "context"

type Repository interface {
	List(ctx context.Context) ([]LoaiMoHinh, error)
}
