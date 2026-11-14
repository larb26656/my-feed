.PHONY: help backend frontend build test test-unit tidy up migrate

help:
	@echo "make up         - start Postgres (docker compose)"
	@echo "make migrate    - apply SQL migrations"
	@echo "make backend    - run Go API (:8080)"
	@echo "make frontend   - run React dev server"
	@echo "make build      - build Go binary"
	@echo "make test       - run all Go tests"
	@echo "make test-unit  - unit tests (no Docker)"
	@echo "make tidy       - go mod tidy"

up:
	cd backend && docker compose up -d

migrate:
	cd backend && ./scripts/migrate.sh

backend:
	cd backend && go run ./cmd/api

frontend:
	cd frontend && npm run dev

build:
	cd backend && go build -o bin/server ./cmd/api

test-unit:
	cd backend && go test ./internal/...

test:
	cd backend && go test ./...

tidy:
	cd backend && go mod tidy
