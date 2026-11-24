package khuvuc

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

type fakeRepository struct {
	called  bool
	floorID string
	items   []KhuVuc
	err     error
}

func (r *fakeRepository) ListByFloorID(_ context.Context, floorID string) ([]KhuVuc, error) {
	r.called = true
	r.floorID = floorID
	return r.items, r.err
}

func (r *fakeRepository) ListShortByFloorID(_ context.Context, floorID string) ([]KhuVucShort, error) {
	r.called = true
	r.floorID = floorID
	shortItems := make([]KhuVucShort, len(r.items))
	for i, item := range r.items {
		shortItems[i] = KhuVucShort{
			IDKhuVuc:     item.IDKhuVuc,
			TenKhuVuc:    item.TenKhuVuc,
			IDLoaiKhuVuc: item.IDLoaiKhuVuc,
		}
	}
	return shortItems, r.err
}

func (r *fakeRepository) ListMonitoring(
	_ context.Context,
	_ string,
	_ MonitoringFilter,
) ([]RoomMonitoring, error) {
	return nil, nil
}

func (r *fakeRepository) ListFloorMonitoring(
	_ context.Context,
	_ string,
) ([]FloorMonitoring, error) {
	return nil, nil
}

func (r *fakeRepository) ListBuildingAreas(
	_ context.Context,
	_ string,
	_ BuildingAreaFilter,
) ([]BuildingArea, int, error) {
	return nil, 0, nil
}

func (r *fakeRepository) ListBuildingFloors(
	_ context.Context,
	_ string,
) ([]BuildingFloor, error) {
	return nil, nil
}

func (r *fakeRepository) ListBuildingAreaTypes(
	_ context.Context,
	_ string,
) ([]BuildingAreaType, error) {
	return nil, nil
}

func TestServiceListByFloorIDReturnsAllAreas(t *testing.T) {
	repository := &fakeRepository{
		items: []KhuVuc{
			{IDKhuVuc: "area-1"},
			{IDKhuVuc: "area-2"},
		},
	}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	items, err := service.ListByFloorID(
		context.Background(),
		" 550e8400-e29b-41d4-a716-446655440000 ",
	)
	if err != nil {
		t.Fatalf("ListByFloorID() error = %v", err)
	}
	if !repository.called || repository.floorID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("repository call = %+v, want trimmed floor ID", repository)
	}
	if len(items) != 2 || items[0].IDKhuVuc != "area-1" || items[1].IDKhuVuc != "area-2" {
		t.Fatalf("areas = %+v, want all repository results", items)
	}
}

func TestServiceListByFloorIDRejectsInvalidUUID(t *testing.T) {
	repository := &fakeRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.ListByFloorID(context.Background(), "not-a-uuid")
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("ListByFloorID() error = %v, want HTTP 400 application error", err)
	}
	if repository.called {
		t.Fatal("repository called with invalid floor ID")
	}
}

func TestServiceListByFloorIDNormalizesEmptyResult(t *testing.T) {
	service, err := NewService(&fakeRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	items, err := service.ListByFloorID(
		context.Background(),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	if err != nil {
		t.Fatalf("ListByFloorID() error = %v", err)
	}
	if items == nil {
		t.Fatal("ListByFloorID() returned nil slice, want empty slice")
	}
	if len(items) != 0 {
		t.Fatalf("len(areas) = %d, want 0", len(items))
	}
}

func TestServiceListByFloorIDPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewService(&fakeRepository{err: wantErr})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.ListByFloorID(
		context.Background(),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListByFloorID() error = %v, want repository error", err)
	}
}
