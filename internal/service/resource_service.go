package service

import (
	"context"

	"github.com/jayant132/seki/internal/domain"
	"github.com/jayant132/seki/internal/repository"
)

type ResourceService struct {
	resources *repository.ResourceRepository
}

func NewResourceService(resources *repository.ResourceRepository) *ResourceService {
	return &ResourceService{resources: resources}
}

func (s *ResourceService) Create(ctx context.Context, name, description string, capacity int) (*domain.Resource, error) {
	return s.resources.Create(ctx, name, description, capacity)
}

func (s *ResourceService) List(ctx context.Context) ([]domain.Resource, error) {
	return s.resources.List(ctx)
}
