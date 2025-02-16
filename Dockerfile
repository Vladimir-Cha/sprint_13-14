# Используем официальный образ Go на основе Alpine
FROM golang:1.23-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Устанавливаем необходимые пакеты для сборки SQLite3
RUN apk update && apk add --no-cache gcc musl-dev

# Копируем go.mod и go.sum для установки зависимостей
COPY project/go.mod project/go.sum ./
RUN go mod download

# Копируем все файлы проекта
COPY project/ .

# Компилируем приложение
RUN go build -o /my_app main.go

# Создаем финальный образ
FROM golang:1.23-alpine

# Устанавливаем рабочую директорию
WORKDIR /app

# Устанавливаем зависимость с sqlite
RUN apk update && apk upgrade
RUN apk add --no-cache sqlite

# Копируем скомпилированное приложение из предыдущего образа
COPY --from=builder /my_app .

# Открываем порт
EXPOSE 7540

ENV TODO_PASSWORD=12345

ENV TODO_DBFILE=.project/data/scheduler.db

# Запускаем приложение
CMD ["./my_app"]
