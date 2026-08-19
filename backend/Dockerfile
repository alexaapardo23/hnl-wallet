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

# Only the two long-running services get built into the image. The
# one-off seed/import programs under cmd/ (seed, seed-tigerbeetle,
# seed-transactions, ...) are ops tooling meant to be run once against a
# fresh environment — see README Seed Data — not containerized services,
# so they're intentionally left out of this image.
RUN go build -o /out/api ./cmd/api
RUN go build -o /out/mcp-server ./cmd/mcp-server

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/api /out/mcp-server ./

# Both binaries ship in one image; docker-compose runs two services from
# it with different `command:` overrides (see docker-compose.yml) rather
# than maintaining two near-identical Dockerfiles.
EXPOSE 8080 8081
CMD ["./api"]
