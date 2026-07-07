# ===== Stage 1: Build =====
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_VERSION=latest
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${APP_VERSION}" \
    -o /domain-list-manager ./cmd/server/

# ===== Stage 2: Final image =====
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

WORKDIR /app

COPY --from=builder /domain-list-manager /domain-list-manager

RUN mkdir -p /app/data /app/output

VOLUME ["/app/data", "/app/output"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

CMD ["/domain-list-manager"]
