package tang

import "context"

type Repository interface {
	ListByBuildingID(
		ctx context.Context,
		buildingID string,
	) ([]Tang, error)
	ListForBuilding(ctx context.Context, buildingID string) ([]Tang, error)
}
