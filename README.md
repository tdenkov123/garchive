# GArchive

gRPC-сервис для управления метаданными файлов с хранением объектов в S3-совместимом хранилище (MinIO), кэшированием в Redis и публикацией доменных событий в Kafka.

## Возможности

- **CreateUpload** — создаёт запись метаданных и возвращает presigned URL для прямой загрузки в MinIO
- **ConfirmUpload** — подтверждает успешную загрузку и переводит файл в статус `ready`
- **GetFile / ListFiles** — чтение метаданных с cursor-пагинацией
- **GetDownloadURL** — presigned URL для скачивания
- **DeleteFile** — soft delete в Postgres + удаление объекта из MinIO + событие в Kafka

## Стек

| Компонент | Технология |
|-----------|------------|
| API | gRPC + Protocol Buffers |
| БД | PostgreSQL 16 |
| Кэш | Redis 7 |
| Object Storage | MinIO (S3 API) |
| Events | Kafka (KRaft) |
| Язык | Go 1.22 |

## Архитектура

```
Client ──gRPC──► FileService
                    │
        ┌───────────┼───────────┐
        ▼           ▼           ▼
   PostgreSQL    Redis       MinIO
   (metadata)   (cache)    (objects)
                    │
                    ▼
                  Kafka
              (file.events)
```

### Поток загрузки файла

1. Клиент вызывает `CreateUpload` → сервис создаёт запись `pending` и presigned PUT URL
2. Клиент загружает файл напрямую в MinIO по presigned URL
3. Клиент вызывает `ConfirmUpload` с SHA-256 → статус `ready`, событие `file.ready` в Kafka

## Быстрый старт

### Требования

- Docker & Docker Compose
- Go 1.22+ (для локальной разработки)
- `protoc` + плагины (для регенерации proto)

### Запуск через Docker

```bash
cp .env.example .env
docker compose up -d --build
```

gRPC-сервер будет доступен на `localhost:50051`.

MinIO Console: http://localhost:9001 (minioadmin / minioadmin)

### Локальная разработка

```bash
# Поднять инфраструктуру
docker compose up -d postgres redis minio kafka

# Запустить сервер
cp .env.example .env
go run ./cmd/server
```

### Пример вызова (grpcurl)

```bash
# Создать upload
grpcurl -plaintext -d '{
  "owner_id": "user-1",
  "original_name": "report.pdf",
  "content_type": "application/pdf",
  "size_bytes": 204800
}' localhost:50051 file.v1.FileService/CreateUpload

# Подтвердить upload
grpcurl -plaintext -d '{
  "id": "<file-id>",
  "owner_id": "user-1",
  "checksum_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}' localhost:50051 file.v1.FileService/ConfirmUpload
```

## Структура проекта

```
cmd/server/          — точка входа
api/proto/           — protobuf-контракты
api/gen/             — сгенерированный Go-код
internal/
  config/            — конфигурация из env
  domain/            — доменные модели и ошибки
  service/           — бизнес-логика
  repository/postgres/
  storage/minio/
  cache/redis/
  events/kafka/
  grpc/              — gRPC handlers
  app/               — wiring / DI
migrations/          — SQL-миграции
```

## Тесты

```bash
go test ./... -race -cover
```

## Makefile

| Команда | Описание |
|---------|----------|
| `make proto` | Регенерация protobuf |
| `make build` | Сборка бинарника |
| `make test` | Запуск тестов |
| `make docker-up` | Docker Compose up |
| `make docker-down` | Docker Compose down |

## Kafka events

Топик `file.events`, формат:

```json
{
  "type": "file.created | file.ready | file.deleted",
  "file_id": "uuid",
  "owner_id": "string",
  "object_key": "owner/uuid/filename",
  "timestamp": "2025-01-15T10:00:00Z"
}
```

## Лицензия

MIT
