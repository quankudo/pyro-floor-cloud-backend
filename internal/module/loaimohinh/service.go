package loaimohinh

import (
	"context"
	"errors"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("model type repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) List(ctx context.Context) ([]LoaiMoHinh, error) {
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []LoaiMoHinh{}
	}
	return items, nil
}
