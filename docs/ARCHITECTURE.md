# Architecture

## Overview

GArchive is a gRPC file metadata service. Clients upload/download objects directly to MinIO via presigned URLs; the service stores metadata in PostgreSQL, caches reads in Redis, and publishes domain events to Kafka.

## C4 — Container diagram

```
[Client] --gRPC/JWT--> [GArchive API]
                           |
         +-----------------+------------------+
         |                 |                  |
   [PostgreSQL]        [Redis]            [MinIO]
         |                                    |
         +----------------> [Kafka] <---------+
```

## Layers

| Layer | Package | Responsibility |
|-------|---------|----------------|
| API | `internal/grpc` | Proto mapping, auth context, error codes |
| Domain | `internal/domain` | Models, invariants |
| Service | `internal/service` | Upload flows, authorization checks |
| Adapters | `repository`, `storage`, `cache`, `events` | I/O |
| Cross-cutting | `auth`, `audit`, `middleware`, `validation` | Security, observability |

## Auth flow (production pattern)

1. Client obtains JWT from IdP (Keycloak / Auth0 / Cognito)
2. Optional API Gateway validates JWT and forwards metadata
3. GArchive interceptor validates JWT again and injects `owner_id` into context
4. Handlers ignore spoofed `owner_id` in request body when JWT is enabled

See [ADR 001](ADR/001-auth-jwt-interceptor.md).

## Testing strategy

| Level | Location | CI job |
|-------|----------|--------|
| Unit | `internal/*/*_test.go` | `test` |
| Integration | `*_integration_test.go` (`-tags=integration`) | `integration` |
| E2e | `tests/e2e/` (`-tags=e2e`) | `e2e` (manual/label) |

## Version audit

Stack versions: [VERSIONS.md](VERSIONS.md) (verified via Context7).
