package thanhvienkhuvuc

import (
	"context"

	"github.com/pyrofloor/cloud-backend/internal/database"
)

type Repository interface {
	AddBatch(ctx context.Context, tx database.DBTX, input AddMembersInput) error
}
