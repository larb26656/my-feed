package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
	"github.com/larb26656/quick-feed/backend/internal/feed/repository"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	Slug        string
	Title       string
	Description string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Feed, error) {
	f, err := domain.NewFeed(in.Slug, in.Title, in.Description)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*domain.Feed, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Publish(ctx context.Context, id uuid.UUID) (*domain.Feed, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := f.Publish(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Service) Archive(ctx context.Context, id uuid.UUID) (*domain.Feed, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := f.Archive(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}
