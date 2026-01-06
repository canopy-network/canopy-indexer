# syntax=docker/dockerfile:1.4

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /workspace

# Install build dependencies (gcc needed for CGO crypto libs)
RUN apk add --no-cache git gcc musl-dev

# Copy canopy dependency first (changes less frequently)
COPY canopy/main ./canopy/main

# Copy go mod files
COPY pgindexer/go.mod pgindexer/go.sum ./pgindexer/

WORKDIR /workspace/pgindexer

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY pgindexer/ ./

# Build with cache mounts for faster rebuilds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/indexer ./cmd/indexer

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /workspace/pgindexer/bin/indexer /app/bin/indexer

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

ENTRYPOINT ["/app/bin/indexer"]
