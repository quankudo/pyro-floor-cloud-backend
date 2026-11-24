package loaikhuvuc

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/pyrofloor/cloud-backend/internal/apperror"
)

type fakeRepository struct {
	called      bool
	modelTypeID string
	items       []LoaiKhuVuc
	err         error
}

func (r *fakeRepository) ListByModelTypeID(
	_ context.Context,
	modelTypeID string,
) ([]LoaiKhuVuc, error) {
	r.called = true
	r.modelTypeID = modelTypeID
	return r.items, r.err
}

func TestServiceListByModelTypeIDReturnsAllAreaTypes(t *testing.T) {
	want := []LoaiKhuVuc{
		{IDLoaiKhuVuc: "area-type-1", IDLoaiMoHinh: "model-1", TenLoaiKhuVuc: "Bedroom"},
		{IDLoaiKhuVuc: "area-type-2", IDLoaiMoHinh: "model-1", TenLoaiKhuVuc: "Kitchen"},
	}
	repository := &fakeRepository{items: want}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	got, err := service.ListByModelTypeID(
		context.Background(),
		" 550e8400-e29b-41d4-a716-446655440000 ",
	)
	if err != nil {
		t.Fatalf("ListByModelTypeID() error = %v", err)
	}
	if !repository.called || repository.modelTypeID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("repository call = %+v, want trimmed model type ID", repository)
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ListByModelTypeID() = %+v, want %+v", got, want)
	}
}

func TestServiceListByModelTypeIDRejectsInvalidUUID(t *testing.T) {
	repository := &fakeRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.ListByModelTypeID(context.Background(), "not-a-uuid")
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("ListByModelTypeID() error = %v, want HTTP 400 application error", err)
	}
	if repository.called {
		t.Fatal("repository called with invalid model type ID")
	}
}

func TestServiceListByModelTypeIDReturnsNonNilEmptySlice(t *testing.T) {
	service, err := NewService(&fakeRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	items, err := service.ListByModelTypeID(
		context.Background(),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	if err != nil {
		t.Fatalf("ListByModelTypeID() error = %v", err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("ListByModelTypeID() = %#v, want non-nil empty slice", items)
	}
}

func TestServiceListByModelTypeIDPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewService(&fakeRepository{err: wantErr})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.ListByModelTypeID(
		context.Background(),
		"550e8400-e29b-41d4-a716-446655440000",
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListByModelTypeID() error = %v, want repository error", err)
	}
}
