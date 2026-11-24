package sos

import "context"

type Repository interface {
	ListCompleted(ctx context.Context, buildingID string, filter ListFilter) ([]SOSReport, int, error)
	ListActive(ctx context.Context, buildingID, userID string, filter ActiveFilter) ([]ActiveSOSReport, int, error)
}
