.PHONY: help backend frontend build test tidy

help:
	@echo "make backend   - run Go API (:8080)"
	@echo "make frontend  - run React dev server"
	@echo "make build     - build Go binary"
	@echo "make test      - run Go tests"
	@echo "make tidy      - go mod tidy"

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

build:
	cd backend && go build -o bin/server ./cmd/server

test:
	cd backend && go test ./...

tidy:
	cd backend && go mod tidy
