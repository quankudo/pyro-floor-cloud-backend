package canhbao

import "context"

type Repository interface {
	List(ctx context.Context, buildingID string, filter ListFilter) ([]AlertListItem, int, error)
	ListActive(ctx context.Context, buildingID, userID string, filter ActiveFilter) ([]ActiveAlertListItem, int, error)
}
