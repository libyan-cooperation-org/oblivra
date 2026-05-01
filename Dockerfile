# OBLIVRA — production container image.
#
# Multi-stage build:
#   1. Build the Svelte frontend into webassets/dist
#   2. Build the Go server with the embedded UI
#   3. Copy a static binary + sigma rules into a minimal runtime image
#
# The final stage is `gcr.io/distroless/static-debian12` so the container
# carries only the binary, the embedded UI, and the seed sigma rules — no
# shell, no package manager, nothing else to attack.

# ---- 1. Frontend build --------------------------------------------------
FROM node:22-alpine AS frontend
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./frontend/
RUN cd frontend && npm ci --no-audit --no-fund
COPY frontend ./frontend
COPY webassets ./webassets
RUN cd frontend && npm run build

# ---- 2. Server build ----------------------------------------------------
FROM golang:1.26-alpine AS server
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/webassets/dist ./webassets/dist
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags "-w -s" -o /out/oblivra-server ./cmd/server \
 && go build -trimpath -ldflags "-w -s" -o /out/oblivra-cli    ./cmd/cli    \
 && go build -trimpath -ldflags "-w -s" -o /out/oblivra-verify ./cmd/verify \
 && go build -trimpath -ldflags "-w -s" -o /out/oblivra-migrate ./cmd/migrate

# ---- 3. Runtime ---------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
USER nonroot:nonroot
WORKDIR /app
COPY --from=server /out/oblivra-server  /app/oblivra-server
COPY --from=server /out/oblivra-cli     /app/oblivra-cli
COPY --from=server /out/oblivra-verify  /app/oblivra-verify
COPY --from=server /out/oblivra-migrate /app/oblivra-migrate
COPY sigma /app/sigma

# Defaults — operators override via env or compose file.
ENV OBLIVRA_DATA_DIR=/var/lib/oblivra \
    OBLIVRA_ADDR=:8080 \
    OBLIVRA_DISABLE_SYSLOG= \
    OBLIVRA_DISABLE_NETFLOW=

EXPOSE 8080/tcp 1514/udp 2055/udp

# /var/lib/oblivra MUST be a mounted volume in production so audit.log,
# cases.log, lineage.log, the WAL, and the BadgerDB hot store survive
# container restarts. Without a volume the audit chain resets every boot.
VOLUME ["/var/lib/oblivra"]

ENTRYPOINT ["/app/oblivra-server"]
