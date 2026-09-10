# --- build stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /seki ./cmd/api

# --- runtime stage ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl && \
    addgroup -S seki && adduser -S seki -G seki

WORKDIR /app

COPY --from=builder /seki /app/seki
COPY migrations ./migrations

USER seki

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD curl -f http://localhost:8080/status/live || exit 1

ENTRYPOINT ["/app/seki"]
