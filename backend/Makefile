.PHONY: help backend build test test-unit check-arch fmt fmt-check lint tidy up down migrate migrate-status migrate-down

GOFUMPT := go run mvdan.cc/gofumpt@v0.9.2

help:
	@echo "make up              - build and start API with persistent SQLite (docker compose)"
	@echo "make down            - stop API (database files remain in data)"
	@echo "make migrate         - apply pending SQL migrations (goose up)"
	@echo "make migrate-status  - show goose migration status"
	@echo "make migrate-down    - roll back one migration (goose down)"
	@echo "make backend         - run Go API (:8080)"
	@echo "make build           - build Go binary"
	@echo "make test            - run all Go tests"
	@echo "make test-unit       - unit tests (no Docker)"
	@echo "make check-arch      - verify Go module and layer dependencies"
	@echo "make fmt             - format backend Go code"
	@echo "make fmt-check       - check backend Go formatting"
	@echo "make lint            - run standard Go linters on the backend"
	@echo "make tidy            - go mod tidy"

up:
	mkdir -p data && docker compose up -d --build

down:
	docker compose down

migrate:
	./scripts/migrate.sh up

migrate-status:
	./scripts/migrate.sh status

migrate-down:
	./scripts/migrate.sh down

dev:
	go run ./cmd/api

build:
	go build -o bin/server ./cmd/api

test-unit:
	go test ./internal/...

test:
	go test ./...

check-arch:
	GOTOOLCHAIN=$$(go env GOVERSION) go run github.com/MouXiaoJun/gomodulith/cmd/gomodulith@v0.7.1 verify --no-cache ./internal/modules/...
	GOTOOLCHAIN=$$(go env GOVERSION) go run github.com/fe3dback/go-arch-lint@v1.19.1 check

fmt:
	$(GOFUMPT) -w .

fmt-check:
	@files=$$($(GOFUMPT) -l .) && \
		if [ -n "$$files" ]; then printf '%s\n' "$$files"; exit 1; fi

lint:
	GOTOOLCHAIN=$$(go env GOVERSION) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 run --no-config --default=standard ./...

tidy:
	go mod tidy
