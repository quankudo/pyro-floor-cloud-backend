package nguoidung

import (
	"context"

	"github.com/pyrofloor/cloud-backend/internal/database"
)

type NguoiDungRepository interface {
	CreateUser(
		ctx context.Context,
		name, email string,
		phone string,
		passwordHash string,
	) error

	FindByID(
		ctx context.Context,
		id string,
	) (*NguoiDung, error)
}

type BatchUserRepository interface {
	ResolveRole(ctx context.Context, roleID, roleCode string) (VaiTro, error)
	WithinTransaction(
		ctx context.Context,
		fn func(database.DBTX) error,
	) error
	FindOrCreateBatchUser(
		ctx context.Context,
		tx database.DBTX,
		user CreateUserInput,
	) (CreatedUser, error)
}

type ResidentRepository interface {
	ListResidents(
		ctx context.Context,
		buildingID string,
		filter ResidentFilter,
	) ([]Resident, int, error)
}

type SupervisorActorRepository interface {
	ListSupervisorActors(
		ctx context.Context,
		buildingID string,
		filter SupervisorActorFilter,
	) ([]SupervisorActor, int, error)
}
