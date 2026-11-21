.PHONY: help backend frontend build test test-unit check-arch tidy up down migrate migrate-status migrate-down

help:
	@echo "make up              - build and start API with persistent SQLite (docker compose)"
	@echo "make down            - stop API (database files remain in backend/data)"
	@echo "make migrate         - apply pending SQL migrations (goose up)"
	@echo "make migrate-status  - show goose migration status"
	@echo "make migrate-down    - roll back one migration (goose down)"
	@echo "make backend         - run Go API (:8080)"
	@echo "make frontend        - run React dev server"
	@echo "make build           - build Go binary"
	@echo "make test            - run all Go tests"
	@echo "make test-unit       - unit tests (no Docker)"
	@echo "make check-arch      - verify Go module and layer dependencies"
	@echo "make tidy            - go mod tidy"

up:
	cd backend && mkdir -p data && docker compose up -d --build

down:
	cd backend && docker compose down

migrate:
	cd backend && ./scripts/migrate.sh up

migrate-status:
	cd backend && ./scripts/migrate.sh status

migrate-down:
	cd backend && ./scripts/migrate.sh down

backend:
	cd backend && go run ./cmd/api

frontend:
	cd frontend && pnpm run dev

build:
	cd backend && go build -o bin/server ./cmd/api

test-unit:
	cd backend && go test ./internal/...

test:
	cd backend && go test ./...

check-arch:
	cd backend && GOTOOLCHAIN=$$(go env GOVERSION) go run github.com/MouXiaoJun/gomodulith/cmd/gomodulith@v0.7.1 verify --no-cache ./internal/modules/...
	cd backend && GOTOOLCHAIN=$$(go env GOVERSION) go run github.com/fe3dback/go-arch-lint@v1.19.1 check

tidy:
	cd backend && go mod tidy
