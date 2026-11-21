package service_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/file/service"
)

func TestStorageSaveReadDelete(t *testing.T) {
	ctx := context.Background()
	storage := service.New(filepath.Join(t.TempDir(), "files"))
	data := []byte("file contents\x00")
	id, err := storage.Save(ctx, data)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("Save returned invalid ID %q: %v", id, err)
	}
	got, err := storage.Read(ctx, id)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("Read = %q, want %q", got, data)
	}
	if err := storage.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := storage.Read(ctx, id); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read after Delete error = %v, want not exist", err)
	}
	if err := storage.Delete(ctx, id); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Delete missing file error = %v, want not exist", err)
	}
}

func TestStorageRejectsInvalidIDs(t *testing.T) {
	ctx := context.Background()
	storage := service.New(t.TempDir())
	for _, id := range []string{"", "not-a-uuid", "../outside"} {
		t.Run(id, func(t *testing.T) {
			if _, err := storage.Read(ctx, id); err == nil {
				t.Error("Read succeeded with invalid ID")
			}
			if err := storage.Delete(ctx, id); err == nil {
				t.Error("Delete succeeded with invalid ID")
			}
		})
	}
}

func TestStorageSaveReturnsDirectoryError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	storage := service.New(filepath.Join(path, "files"))
	id, err := storage.Save(context.Background(), []byte("data"))
	if err == nil {
		t.Fatal("Save succeeded with a file in its directory path")
	}
	if id != "" {
		t.Fatalf("Save returned ID %q on error", id)
	}
}
