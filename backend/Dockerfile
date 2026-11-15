FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/api \
    && CGO_ENABLED=0 go build -o /out/goose github.com/pressly/goose/v3/cmd/goose

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server /usr/local/bin/server
COPY --from=build /out/goose /usr/local/bin/goose
COPY migrations /app/migrations
RUN mkdir -p /data
ENV DATABASE_PATH=/data/quick-feed.db HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/bin/sh", "-c", "/usr/local/bin/goose -dir /app/migrations sqlite3 \"$DATABASE_PATH\" up && exec /usr/local/bin/server"]
