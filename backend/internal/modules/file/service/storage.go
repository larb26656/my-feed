package service

import (
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type Storage struct{ dir string }

func New(dir string) *Storage { return &Storage{dir: dir} }
func (s *Storage) Save(_ context.Context, data []byte) (string, error) {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", err
	}
	id := uuid.NewString()
	if err := os.WriteFile(filepath.Join(s.dir, id), data, 0o600); err != nil {
		_ = os.Remove(filepath.Join(s.dir, id))
		return "", err
	}
	return id, nil
}

func (s *Storage) Read(_ context.Context, id string) ([]byte, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(s.dir, id))
}

func (s *Storage) Delete(_ context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return err
	}
	return os.Remove(filepath.Join(s.dir, id))
}
