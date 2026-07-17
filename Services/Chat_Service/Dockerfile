# Sozvon\Services\Chat_Service\Dockerfile

# ─── Stage 1: build ───────────────────────────────────────────────────────────
FROM golang:1.26.4-alpine AS builder

# Нужны git + ca-certificates для go mod download
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Сначала копируем только зависимости — кэш слоёв не сбрасывается при изменении кода
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 — статический бинарь без libc (работает в scratch/alpine)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/service ./main.go

# ─── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/service .
# COPY --from=builder /app/Media ./Media
COPY --from=builder /app/db ./db

# Порт переопределяется через ENV в docker-compose
EXPOSE 8080

ENTRYPOINT ["./service"]