package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

type Source struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewSource(name string) (*Source, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apperr.Validation("name is required")
	}
	now := time.Now().UTC()
	return &Source{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
