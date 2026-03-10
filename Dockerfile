# Build Stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for go-duckdb plugin support
RUN apk add --no-cache gcc g++ make pkgconfig

WORKDIR /app

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the headless server application
# We disable CGO for pure Go if possible, but DuckDB might need it depending on OS.
# Since we are deploying backend-only, we skip Wails entirely.
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-s -w" -o /app/bin/sovereign-server .
RUN go build -ldflags="-s -w" -o /app/bin/oblivra-cli ./cmd/cli

# Runtime Stage
FROM alpine:latest

# Install runtime dependencies (e.g. ca-certificates)
RUN apk add --no-cache ca-certificates tzdata

# Domain 6 Hardening: Create a non-privileged user and switch to it
RUN addgroup -S oblivragrp && adduser -S oblivra -G oblivragrp

WORKDIR /app

# Ensure configuration and data directories exist with strict ownership
RUN mkdir -p /var/lib/oblivrashell/plugins \
    && mkdir -p /var/lib/oblivrashell/db \
    && mkdir -p /var/lib/oblivrashell/logs \
    && chown -R oblivra:oblivragrp /var/lib/oblivrashell \
    && chown -R oblivra:oblivragrp /app

# Copy binaries from the builder stage
COPY --chown=oblivra:oblivragrp --from=builder /app/bin/sovereign-server /app/sovereign-server
COPY --chown=oblivra:oblivragrp --from=builder /app/bin/oblivra-cli /app/oblivra-cli
COPY --chown=oblivra:oblivragrp --from=builder /app/plugins /var/lib/oblivrashell/plugins

# Drop root privileges
USER oblivra:oblivragrp

# Set environment variables for the new paths
ENV OBLIVRA_DATA_DIR=/var/lib/oblivrashell

# Expose API/WebSocket port
EXPOSE 8080
# Expose Raft cluster port
EXPOSE 8443

# Start the headless server
CMD ["/app/sovereign-server"]
