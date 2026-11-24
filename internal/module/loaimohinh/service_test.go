package loaimohinh

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	called bool
	items  []LoaiMoHinh
	err    error
}

func (r *fakeRepository) List(context.Context) ([]LoaiMoHinh, error) {
	r.called = true
	return r.items, r.err
}

func TestServiceListReturnsAllModelTypes(t *testing.T) {
	want := []LoaiMoHinh{
		{IDLoaiMoHinh: "model-1", TenLoaiMoHinh: "Apartment"},
		{IDLoaiMoHinh: "model-2", TenLoaiMoHinh: "House"},
	}
	repository := &fakeRepository{items: want}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !repository.called {
		t.Fatal("repository List() was not called")
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
}

func TestServiceListReturnsNonNilEmptySlice(t *testing.T) {
	service, err := NewService(&fakeRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("List() = %#v, want non-nil empty slice", items)
	}
}

func TestServiceListPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewService(&fakeRepository{err: wantErr})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.List(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("List() error = %v, want repository error", err)
	}
}
