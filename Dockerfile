# Стадия сборки
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Сначала зависимости
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Затем весь исходный код
COPY . .

# Собираем бинарник из cmd/app/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app ./cmd/app

# Рантайм-стадия
FROM alpine:latest

WORKDIR /app

# Необязательный, но полезный пакет для таймзон и т.п.
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/app .
COPY --from=builder /app/templates ./templates
# Скопируй конфиг если нужен
# COPY config/config.yaml ./config.yaml

EXPOSE 8080

CMD ["./app"]
