package khuvuc

import "context"

type Repository interface {
	ListByFloorID(ctx context.Context, floorID string) ([]KhuVuc, error)
	ListShortByFloorID(ctx context.Context, floorID string) ([]KhuVucShort, error)
	ListMonitoring(ctx context.Context, buildingID string, filter MonitoringFilter) ([]RoomMonitoring, error)
	ListFloorMonitoring(ctx context.Context, buildingID string) ([]FloorMonitoring, error)
	ListBuildingAreas(ctx context.Context, buildingID string, filter BuildingAreaFilter) ([]BuildingArea, int, error)
	ListBuildingFloors(ctx context.Context, buildingID string) ([]BuildingFloor, error)
	ListBuildingAreaTypes(ctx context.Context, buildingID string) ([]BuildingAreaType, error)
}
