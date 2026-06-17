.PHONY: proto generate build test lint run docker-up docker-down

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
	go test ./... -count=1 -race -cover

lint:
	golangci-lint run ./...

run: build
	./bin/server

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v
