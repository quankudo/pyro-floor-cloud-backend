package thanhvienkhuvuc

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/pyrofloor/cloud-backend/internal/apperror"
	"github.com/pyrofloor/cloud-backend/internal/database"
)

var (
	ErrTargetNotFound = apperror.New(
		http.StatusNotFound,
		apperror.AppErrorNotFound,
		"building or area was not found in the requested building",
		nil,
	)
	ErrAlreadyMember = apperror.New(
		http.StatusConflict,
		apperror.AppErrorConflict,
		"user already has a membership for this area",
		nil,
	)
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("area membership repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) AddBatch(
	ctx context.Context,
	tx database.DBTX,
	input AddMembersInput,
) error {
	if _, err := uuid.Parse(input.BuildingID); err != nil {
		return apperror.New(
			http.StatusBadRequest,
			apperror.AppErrorInvalidArgument,
			"building ID must be a UUID",
			nil,
		)
	}
	if len(input.Members) == 0 {
		return nil
	}
	for _, member := range input.Members {
		if _, err := uuid.Parse(member.UserID); err != nil {
			return apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"user ID must be a UUID",
				nil,
			)
		}
		if _, err := uuid.Parse(member.AreaID); err != nil {
			return apperror.New(
				http.StatusBadRequest,
				apperror.AppErrorInvalidArgument,
				"area ID must be a UUID",
				nil,
			)
		}
	}
	if tx == nil {
		return errors.New("database transaction is required")
	}
	return s.repository.AddBatch(ctx, tx, input)
}
