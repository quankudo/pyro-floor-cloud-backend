package phanconggiamsat

import (
	"context"

	"github.com/pyrofloor/cloud-backend/internal/database"
)

type Repository interface {
	AddBatch(ctx context.Context, tx database.DBTX, input AddAssignmentsInput) error
	ListAssignmentScopes(ctx context.Context, buildingID string, filter AssignmentScopeFilter) ([]AssignmentScope, error)
}
