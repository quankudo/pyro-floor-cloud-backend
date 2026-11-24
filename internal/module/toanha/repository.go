package toanha

import (
	"context"

	"github.com/pyrofloor/cloud-backend/internal/database"
)

type ToaNhaRepository interface {
	FindByID(context context.Context, id string) (*ToaNha, error)
	List(context context.Context, filter ListFilter) ([]ToaNha, int64, error)
	FindOwnerID(ctx context.Context, tx database.DBTX, buildingID string) (string, error)
}
