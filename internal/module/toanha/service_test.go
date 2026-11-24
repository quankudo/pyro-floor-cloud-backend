package toanha

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/database"
)

type fakeToaNhaRepository struct {
	filter     ListFilter
	items      []ToaNha
	total      int64
	err        error
	called     bool
	owner      string
	ownerErr   error
	ownerCalls int
}

func (r *fakeToaNhaRepository) FindByID(context.Context, string) (*ToaNha, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeToaNhaRepository) List(_ context.Context, filter ListFilter) ([]ToaNha, int64, error) {
	r.called = true
	r.filter = filter
	return r.items, r.total, r.err
}

func (r *fakeToaNhaRepository) FindOwnerID(
	_ context.Context,
	_ database.DBTX,
	_ string,
) (string, error) {
	r.ownerCalls++
	return r.owner, r.ownerErr
}

type fakeDBTX struct{}

func (fakeDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (fakeDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (fakeDBTX) QueryRow(context.Context, string, ...any) pgx.Row {
	return nil
}

func TestToaNhaServiceListAppliesDefaultsAndNormalizesFilters(t *testing.T) {
	repository := &fakeToaNhaRepository{
		items: []ToaNha{{IDToaNha: "building-1"}},
		total: 1,
	}
	service := NewToaNhaService(repository)

	page, err := service.List(context.Background(), ListFilter{
		TrangThai:    " HoatDong ",
		IDLoaiMoHinh: " 550e8400-e29b-41d4-a716-446655440000 ",
		TenToaNha:    "  Chung cu  ",
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !repository.called {
		t.Fatal("repository List() was not called")
	}
	if repository.filter.Limit != DefaultPageSize ||
		repository.filter.Offset != 0 ||
		repository.filter.TrangThai != "HoatDong" ||
		repository.filter.IDLoaiMoHinh != "550e8400-e29b-41d4-a716-446655440000" ||
		repository.filter.TenToaNha != "Chung cu" {
		t.Fatalf("repository filter = %+v, want normalized values and defaults", repository.filter)
	}
	if page.Limit != DefaultPageSize || page.Offset != 0 || page.Total != 1 ||
		len(page.Items) != 1 || page.Items[0].IDToaNha != "building-1" {
		t.Fatalf("unexpected page: %+v", page)
	}
}

func TestToaNhaServiceListRejectsInvalidFiltersBeforeRepository(t *testing.T) {
	tests := []struct {
		name   string
		filter ListFilter
	}{
		{name: "negative limit", filter: ListFilter{Limit: -1}},
		{name: "limit too large", filter: ListFilter{Limit: MaxPageSize + 1}},
		{name: "negative offset", filter: ListFilter{Offset: -1}},
		{name: "invalid model UUID", filter: ListFilter{IDLoaiMoHinh: "not-a-uuid"}},
		{name: "status too long", filter: ListFilter{TrangThai: string(make([]byte, 21))}},
		{name: "name too long", filter: ListFilter{TenToaNha: string(make([]byte, 151))}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeToaNhaRepository{}
			_, err := NewToaNhaService(repository).List(context.Background(), tt.filter)
			if err == nil {
				t.Fatal("List() error = nil, want validation error")
			}
			if !apperror.IsCode(err, apperror.AppErrorInvalidArgument) {
				t.Fatalf("error code is not invalid argument: %v", err)
			}
			if repository.called {
				t.Fatal("repository List() called for invalid filter")
			}
			var appErr *apperror.Error
			if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadRequest {
				t.Fatalf("error = %v, want HTTP 400 application error", err)
			}
		})
	}
}

func TestToaNhaServiceListPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repository := &fakeToaNhaRepository{err: wantErr}

	_, err := NewToaNhaService(repository).List(context.Background(), ListFilter{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("List() error = %v, want wrapped repository error", err)
	}
}

func TestToaNhaServiceEnsureOwnerChecksOwnerOnce(t *testing.T) {
	repository := &fakeToaNhaRepository{owner: "550e8400-e29b-41d4-a716-446655440002"}
	service := NewToaNhaService(repository)

	err := service.EnsureOwner(
		context.Background(),
		fakeDBTX{},
		"550e8400-e29b-41d4-a716-446655440030",
		repository.owner,
	)
	if err != nil {
		t.Fatalf("EnsureOwner() error = %v", err)
	}
	if repository.ownerCalls != 1 {
		t.Fatalf("FindOwnerID() calls = %d, want 1", repository.ownerCalls)
	}
}

func TestToaNhaServiceEnsureOwnerRejectsDifferentOwner(t *testing.T) {
	repository := &fakeToaNhaRepository{owner: "550e8400-e29b-41d4-a716-446655440003"}
	service := NewToaNhaService(repository)

	err := service.EnsureOwner(
		context.Background(),
		fakeDBTX{},
		"550e8400-e29b-41d4-a716-446655440030",
		"550e8400-e29b-41d4-a716-446655440002",
	)
	if !apperror.IsCode(err, apperror.AppErrorForbidden) {
		t.Fatalf("EnsureOwner() error = %v, want forbidden", err)
	}
}
