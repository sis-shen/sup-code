# ── Build stage ──────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/supcode ./cmd/supcode

# ── Runtime stage ────────────────────────────────────────
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S supcode && adduser -S supcode -G supcode

WORKDIR /home/supcode

COPY --from=builder /out/supcode /usr/local/bin/supcode
COPY packaging/docker/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && chown supcode:supcode /usr/local/bin/supcode

USER supcode

ENTRYPOINT ["/entrypoint.sh"]
CMD ["--help"]

LABEL org.opencontainers.image.title="SupCode" \
      org.opencontainers.image.description="AI coding agent with multi-agent collaboration" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/supcode/supcode"
