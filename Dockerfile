# Dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -o smsleopard-api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -o smsleopard-worker ./cmd/worker

# Build Script binaries (NEW)
RUN CGO_ENABLED=0 GOOS=linux go build -o smsleopard-migrate ./scripts/migrate.go
RUN CGO_ENABLED=0 GOOS=linux go build -o smsleopard-seed ./scripts/seed.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates wget
WORKDIR /app

# Copy all binaries
COPY --from=builder /app/smsleopard-api .
COPY --from=builder /app/smsleopard-worker .
COPY --from=builder /app/smsleopard-migrate .   
COPY --from=builder /app/smsleopard-seed .       

# Copy .env file for configuration
COPY .env .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

# Default to API, overridden by docker-compose for worker
CMD ["./smsleopard-api"]