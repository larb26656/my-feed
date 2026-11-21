package api

import "context"

type Storage interface {
	Save(context.Context, []byte) (string, error)
	Read(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}
