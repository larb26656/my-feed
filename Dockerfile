FROM node:22-alpine AS frontend-build
RUN corepack enable && corepack prepare pnpm@10.28.2 --activate
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM golang:1.25-alpine AS backend-build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/api \
    && GOBIN=/out CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@$(go list -m -f '{{.Version}}' github.com/pressly/goose/v3)

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-build /out/server /usr/local/bin/server
COPY --from=backend-build /out/goose /usr/local/bin/goose
COPY --from=frontend-build /src/frontend/dist /app/web
COPY backend/migrations /app/migrations
RUN mkdir -p /data
ENV DATABASE_PATH=/data/my-feed.db HTTP_ADDR=:8080 WEB_ROOT=/app/web
EXPOSE 8080
ENTRYPOINT ["/bin/sh", "-c", "/usr/local/bin/goose -dir /app/migrations sqlite3 \"$DATABASE_PATH\" up && exec /usr/local/bin/server"]
