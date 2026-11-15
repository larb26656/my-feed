package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const (
	busyTimeoutMS = 5000
	maxOpenConns  = 1
)

// Transaction is an opaque transaction handle shared by application ports.
// Database adapters interpret the handle for their own driver.
type Transaction interface{}

// Transactor runs application work atomically without exposing driver types.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, tx Transaction) error) error
}

// Pool owns the single SQLite connection used by this backend instance.
type Pool struct {
	*sql.DB
}

// NewPool opens a persistent SQLite database at path. SQLite serializes writes;
// a single connection also queues in-process reads and writes behind transactions.
func NewPool(ctx context.Context, path string) (*Pool, error) {
	if path == "" {
		return nil, fmt.Errorf("database path must not be empty")
	}
	if path != ":memory:" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve database path: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
		path = absPath
	}

	dsn := sqliteDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect SQLite database: %w", err)
	}
	return &Pool{DB: db}, nil
}

func sqliteDSN(path string) string {
	if path == ":memory:" {
		return fmt.Sprintf("file::memory:?_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)&_txlock=immediate", busyTimeoutMS)
	}
	return (&url.URL{
		Scheme:   "file",
		Path:     path,
		RawQuery: fmt.Sprintf("_pragma=busy_timeout(%d)&_pragma=foreign_keys(1)&_txlock=immediate", busyTimeoutMS),
	}).String()
}

func (p *Pool) WithinTransaction(ctx context.Context, fn func(ctx context.Context, tx Transaction) error) error {
	tx, err := p.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
