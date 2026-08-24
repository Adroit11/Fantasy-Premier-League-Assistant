# ==============================================================================
# Stage 1: Build
# ==============================================================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependencies separately from source code
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -extldflags '-static'" \
    -o /app/fpl-assistant ./cmd/server

# ==============================================================================
# Stage 2: Runtime
# ==============================================================================
FROM scratch

# Pull in time-zone data and root CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binary
COPY --from=builder /app/fpl-assistant /fpl-assistant

# Expose the default backend port
EXPOSE 18492

# Default bind to all interfaces so Docker port mapping works.
# Override BIND_HOST=127.0.0.1 for local dev outside Docker.
ENV BIND_HOST=0.0.0.0

ENTRYPOINT ["/fpl-assistant"]
