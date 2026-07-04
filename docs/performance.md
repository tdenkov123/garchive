# Performance baseline

Run locally with stack up:

```bash
go install github.com/bojand/ghz/cmd/ghz@latest
ghz --insecure \
  -n 1000 -c 10 \
  -d '{"owner_id":"user-1","original_name":"bench.bin","content_type":"application/octet-stream","size_bytes":1048576}' \
  localhost:50051 file.v1.FileService/CreateUpload
```

Record results here after each major release.

| RPC | RPS | p50 | p99 | Date |
|-----|-----|-----|-----|------|
| CreateUpload | TBD | TBD | TBD | 2026-07-04 |
