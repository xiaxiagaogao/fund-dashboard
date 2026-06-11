# Multi-stage image for fund-dashboard.
#
#   Stage 1: build the SvelteKit SPA (node:22-alpine)
#   Stage 2: build the Go binary (golang:1.23-alpine)
#   Stage 3: minimal runtime (alpine + binary + SPA build + ca-certs)
#
# The runtime container has NO build tooling, no source. Final size ~30MB.
#
# Build:  docker build -t fund-dashboard:latest .
# Run:    docker run --rm -p 8090:8090 -v /path/to/data:/app/data --env-file .env fund-dashboard:latest

# ---- Frontend build ------------------------------------------------------
FROM node:22-alpine AS frontend
WORKDIR /web
# Copy only package.json (NOT lockfile) — the lockfile generated on the
# developer's host (macOS/arm) pins platform-specific rollup natives that
# don't exist for linux/amd64. Regenerating in-container picks the right
# `@rollup/rollup-linux-x64-musl` etc.
COPY web/package.json ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---- Backend build -------------------------------------------------------
FROM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build static binaries (no CGO — modernc.org/sqlite is pure Go)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dashboard ./cmd/dashboard \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dashctl ./cmd/dashctl

# ---- Runtime -------------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /out/dashboard /usr/local/bin/dashboard
COPY --from=backend /out/dashctl /usr/local/bin/dashctl
COPY --from=frontend /web/build /app/web_build
EXPOSE 8090
ENV STATIC_DIR=/app/web_build
ENV FUND_DB_PATH=/app/data/fund.db
ENV HTTP_ADDR=:8090
# Persistent fund.db lives on a bind-mount or named volume at /app/data
VOLUME ["/app/data"]
ENTRYPOINT ["/usr/local/bin/dashboard"]
