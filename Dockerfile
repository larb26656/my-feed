FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-build
RUN corepack enable && corepack prepare pnpm@10.28.2 --activate
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go install github.com/pressly/goose/v3/cmd/goose@$(go list -m -f '{{.Version}}' github.com/pressly/goose/v3) \
    && mkdir -p /out \
    && find "$(go env GOPATH)/bin" -type f -name goose -exec cp {} /out/goose \; \
    && test -x /out/goose
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/server ./cmd/api

FROM --platform=$BUILDPLATFORM alpine:3.22 AS runtime-files
RUN apk add --no-cache ca-certificates && mkdir -p /data

FROM alpine:3.22
WORKDIR /app
COPY --from=runtime-files /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=runtime-files /data /data
COPY --from=backend-build /out/server /usr/local/bin/server
COPY --from=backend-build /out/goose /usr/local/bin/goose
COPY --from=frontend-build /src/frontend/dist /app/web
COPY backend/migrations /app/migrations
ENV DATABASE_PATH=/data/my-feed.db HTTP_ADDR=:8080 WEB_ROOT=/app/web
EXPOSE 8080
ENTRYPOINT ["/bin/sh", "-c", "/usr/local/bin/goose -dir /app/migrations sqlite3 \"$DATABASE_PATH\" up && exec /usr/local/bin/server"]
