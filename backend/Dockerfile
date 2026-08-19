# syntax=docker/dockerfile:1

# TigerBeetle's Go client (github.com/tigerbeetle/tigerbeetle-go) links a
# prebuilt native static library via CGO — see its native/ directory. That
# means:
#   - CGO_ENABLED must stay 1 (the usual CGO_ENABLED=0 "static Go binary"
#     trick would break the TigerBeetle bindings).
#   - The builder needs a C toolchain (gcc), which the plain golang image
#     doesn't include.
#   - The runtime base should share the builder's libc family (glibc) —
#     switching to Alpine (musl) here risks a runtime linking mismatch for
#     the same reason CGO binaries generally aren't portable across libcs.
# Both stages are Debian-based for that reason, not Alpine.

FROM golang:1.25-bookworm AS builder

ENV CGO_ENABLED=1

RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# The two long-running services, plus the one-off seed/import programs
# under cmd/ — those run once, orchestrated by scripts/seed-all.sh, as
# docker-compose's one-shot `seed` service (see README Seed Data). They
# ship in this same image rather than a separate one to avoid maintaining
# a second Dockerfile for five small Go binaries.
RUN go build -o /out/api ./cmd/api
RUN go build -o /out/mcp-server ./cmd/mcp-server
RUN go build -o /out/seed ./cmd/seed
RUN go build -o /out/seed-tigerbeetle ./cmd/seed-tigerbeetle
RUN go build -o /out/seed-initial-balances ./cmd/seed-initial-balances
RUN go build -o /out/seed-transactions ./cmd/seed-transactions
RUN go build -o /out/seed-guard ./cmd/seed-guard

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/api /out/mcp-server /out/seed /out/seed-tigerbeetle /out/seed-initial-balances /out/seed-transactions /out/seed-guard ./
COPY scripts/seed-all.sh ./
RUN chmod +x ./seed-all.sh

# All binaries ship in one image; docker-compose runs different services
# from it with different `command:` overrides (see docker-compose.yml)
# rather than maintaining several near-identical Dockerfiles.
EXPOSE 8080 8081
CMD ["./api"]
