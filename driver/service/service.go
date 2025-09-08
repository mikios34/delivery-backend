package service

import (
	"context"

	repo "github.com/mikios34/delivery-backend/driver/repository"
	"github.com/mikios34/delivery-backend/models"
)

type Service struct {
	repo *repo.Repository
}

func NewService(r *repo.Repository) *Service {
	return &Service{repo: r}
}

// NewServiceImpl is an explicit implementation constructor (alias).
func NewServiceImpl(r *repo.Repository) *Service {
	return NewService(r)
}

func (s *Service) ListDrivers(ctx context.Context) ([]models.Driver, error) {
	return s.repo.ListDrivers(ctx)
}

func (s *Service) GetDriver(ctx context.Context, id uint) (*models.Driver, error) {
	return s.repo.GetDriverByID(ctx, id)
}

func (s *Service) CreateDriver(ctx context.Context, d *models.Driver) (*models.Driver, error) {
	// business rules could go here
	return s.repo.StoreDriver(ctx, d)
}

func (s *Service) DeleteDriver(ctx context.Context, id uint) error {
	// business rules / soft-delete checks could go here
	return s.repo.DeleteDriver(ctx, id)
}
