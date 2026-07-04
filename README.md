# GArchive

[![CI](https://github.com/tdenkov123/garchive/actions/workflows/ci.yml/badge.svg)](https://github.com/tdenkov123/garchive/actions/workflows/ci.yml)

gRPC-сервис для управления метаданными файлов с хранением объектов в S3-совместимом хранилище (MinIO), кэшированием в Redis и публикацией доменных событий в Kafka.

## Возможности

### Single upload
- **CreateUpload** — метаданные + presigned PUT URL
- **ConfirmUpload** — SHA-256 checksum → статус `ready` + Kafka `file.ready`

### Multipart upload
- **CreateMultipartUpload**, **GetPartUploadURL**, **ReportPartUploaded**
- **ListUploadParts**, **CompleteMultipartUpload**, **AbortMultipartUpload**

### Read / delete
- **GetFile / ListFiles** — cursor-пагинация
- **GetDownloadURL** — presigned download
- **DeleteFile** — soft delete + удаление объекта + Kafka event

### Security & ops
- JWT auth (HS256 interceptor), optional TLS
- Rate limiting, input validation, audit logging
- Prometheus metrics (`:9090/metrics`), gRPC health

## Стек

| Компонент | Версия |
|-----------|--------|
| Go | 1.26 |
| gRPC | google.golang.org/grpc v1.82.0 |
| pgx | github.com/jackc/pgx/v5 v5.10.0 |
| go-redis | github.com/redis/go-redis/v9 v9.7.3 |
| PostgreSQL | 18.4-alpine |
| Redis | 8.8.0-alpine |
| MinIO | RELEASE.2025-09-07T16-13-09Z |
| Kafka | 4.3.1 (KRaft) |

## Быстрый старт

```bash
cp .env.example .env
docker compose up -d --build
```

gRPC: `localhost:50051` · Metrics: `localhost:9090/metrics` · MinIO Console: http://localhost:9001

### Authenticated grpcurl (JWT)

```bash
export JWT_HMAC_SECRET=change-me-in-production
TOKEN=$(bash scripts/get-token.sh user-1)

grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"owner_id":"user-1","original_name":"doc.pdf","content_type":"application/pdf","size_bytes":1024}' \
  localhost:50051 file.v1.FileService/CreateUpload
```

Enable JWT: `JWT_ENABLED=true` in `.env`.

### Compose profiles

```bash
docker compose --profile auth up -d keycloak    # OIDC demo IdP :8080
docker compose --profile observability up -d jaeger
```

Production layout: [docker-compose.prod.yml](docker-compose.prod.yml)

## Testing strategy

```bash
make test              # unit + race + coverage
make test-integration  # testcontainers (Docker required)
make test-e2e          # full upload flows
make lint              # golangci-lint
```

| Level | Coverage target |
|-------|-----------------|
| Unit | ≥ 40% (CI gate 30%) |
| Integration | postgres, redis, minio, kafka |
| E2e | single + multipart + auth |

## Security checklist

See [docs/SECURITY.md](docs/SECURITY.md).

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Security](docs/SECURITY.md)
- [Versions / Context7 audit](docs/VERSIONS.md)
- [ADRs](docs/ADR/)
- [CHANGELOG](CHANGELOG.md)

## License
- [MIT](LICENSE)
