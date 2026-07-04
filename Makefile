.PHONY: proto generate build test test-integration test-e2e lint run docker-up docker-down coverage

PROTO_DIR=api/proto
GEN_DIR=api/gen

proto:
	@mkdir -p $(GEN_DIR)
	protoc \
		-I $(PROTO_DIR) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/file/v1/file.proto

generate: proto

build:
	go build -o bin/server ./cmd/server

test:
	CGO_ENABLED=1 go test ./... -count=1 -race -coverprofile=coverage.out -covermode=atomic -coverpkg=./...

coverage: test
	go tool cover -html=coverage.out -o coverage.html

test-integration:
	CGO_ENABLED=1 go test -tags=integration ./... -race -count=1 -timeout 10m

test-e2e:
	CGO_ENABLED=1 go test -tags=e2e ./tests/e2e/... -race -count=1 -timeout 15m

lint:
	golangci-lint run ./...

run: build
	./bin/server

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v
