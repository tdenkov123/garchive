# Security

## Threat model (STRIDE-lite)

| Threat | Mitigation |
|--------|------------|
| Spoofed owner_id | JWT interceptor; owner from verified claims |
| Plaintext transport | TLS on gRPC (`GRPC_INSECURE=false`) |
| Default credentials in prod | `APP_ENV=production` startup guard |
| Oversized uploads | `MAX_FILE_SIZE_BYTES` validation |
| Path traversal in object keys | `owner_id` format validation before `path.Join` |
| API enumeration | Disable gRPC reflection in production |
| Dependency CVEs | `govulncheck`, Dependabot, Trivy in CI |
| Container as root | Non-root `USER 65534` in Dockerfile |

## Security checklist

- [ ] Set `APP_ENV=production`
- [ ] Set strong `JWT_HMAC_SECRET` and enable `JWT_ENABLED=true`
- [ ] Provide TLS certs; set `GRPC_INSECURE=false`
- [ ] Set `GRPC_ENABLE_REFLECTION=false`
- [ ] Replace default Postgres/MinIO credentials
- [ ] Use `docker-compose.prod.yml` (internal network)
- [ ] Run `govulncheck ./...` and Trivy scan before deploy

## Audit events

Structured logs (`component=audit`):

- `auth.denied`
- `file.created`
- `upload.confirmed`
- `file.deleted`

## Reporting

Report security issues privately to the repository owner.
