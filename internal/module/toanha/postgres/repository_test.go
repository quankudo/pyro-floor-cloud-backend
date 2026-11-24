package postgres

import (
	"reflect"
	"testing"

	"github.com/pyrofloor/cloud-backend/internal/module/toanha"
)

func TestBuildListWhereAddsOnlyConfiguredFilters(t *testing.T) {
	filter := toanha.ListFilter{
		TrangThai:    "HoatDong",
		IDLoaiMoHinh: "550e8400-e29b-41d4-a716-446655440000",
		TenToaNha:    "A%' OR TRUE --",
	}

	where, args := buildListWhere(filter)
	wantWhere := " WHERE trang_thai = $1 AND id_loai_mo_hinh = $2 AND strpos(lower(ten_toa_nha), lower($3)) > 0"
	wantArgs := []any{"HoatDong", "550e8400-e29b-41d4-a716-446655440000", "A%' OR TRUE --"}
	if where != wantWhere || !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("buildListWhere() = (%q, %#v), want (%q, %#v)", where, args, wantWhere, wantArgs)
	}
}

func TestBuildListWhereWithoutFilters(t *testing.T) {
	where, args := buildListWhere(toanha.ListFilter{})
	if where != "" || len(args) != 0 {
		t.Fatalf("buildListWhere() = (%q, %#v), want empty filter", where, args)
	}
}
