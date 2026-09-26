# syntax=docker/dockerfile:1

FROM golang:1.27-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS builder

WORKDIR /src

ENV CGO_ENABLED=0 \
    GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY api/gen/ api/gen/

ARG TARGETOS=linux
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

WORKDIR /app

COPY --from=builder /out/server /app/server

USER nonroot

ENV APP_TMP_DATA=/tmp \
    TZ=UTC

EXPOSE 50051 9090

LABEL org.opencontainers.image.title="garchive" \
      org.opencontainers.image.description="gRPC file metadata service" \
      org.opencontainers.image.source="https://github.com/tdenkov123/garchive"

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["/app/server", "health"]

ENTRYPOINT ["/app/server"]
