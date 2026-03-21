# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

ARG SERVICE
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy workspace descriptor and all modules
COPY go.work go.work.sum ./
COPY gen/ gen/
COPY services/ services/

# Build the target service
WORKDIR /src/services/${SERVICE}
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /server /server

ENTRYPOINT ["/server"]
