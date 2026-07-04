# Stack Versions

Last Context7 audit: **2026-07-04**

| Component | Version | CVE / advisory | Min safe | Source |
|-----------|---------|----------------|----------|--------|
| pgx | v5.10.0 | GO-2026-5004 | ≥ v5.9.2 | Context7 `/jackc/pgx`, govulncheck |
| gRPC | v1.82.0 | GO-2026-4762 | ≥ v1.79.3 | Context7 `/grpc/grpc-go`, govulncheck |
| go-redis | v9.7.3 | GO-2025-3540 | ≥ v9.7.3 | Context7 `/redis/go-redis`, govulncheck |
| kafka-go | v0.4.51 | — | latest patch | go list -u |
| golang.org/x/net | v0.55.0 | GO-2026-5026+ | ≥ v0.55.0 | govulncheck |
| golang.org/x/crypto | v0.52.0 | GO-2026-5033 | ≥ v0.52.0 | govulncheck |
| golang.org/x/sys | v0.45.0 | GO-2026-5024 | ≥ v0.44.0 | govulncheck |
| JWT | v5.3.1 | — | latest | go.mod |
| Go | 1.26 | — | — | go.mod, CI |

## Context7 library IDs

| Library | ID |
|---------|-----|
| pgx | `/jackc/pgx` |
| gRPC Go | `/grpc/grpc-go` |
| go-redis | `/redis/go-redis` |

Verify before upgrades: `govulncheck ./...` and Context7 `query-docs` for breaking changes.
