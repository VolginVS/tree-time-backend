# Этап 1: Сборка бинарника
FROM golang:1.21-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o /app/server ./cmd/server

# Этап 2: Финальный образ
FROM alpine:latest

# Копируем собранный бинарник
COPY --from=builder /app/server /server

# Открываем порт (замените на ваш)
EXPOSE 8081

# Точка входа
ENTRYPOINT ["/server"]

# Параметры по умолчанию (можно переопределить)
CMD ["--port=8081"]