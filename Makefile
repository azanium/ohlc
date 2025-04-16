.PHONY: build run clean proto test test-coverage

GO=go
PROTOC=protoc
PROTO_DIR=proto
BIN_DIR=bin
BIN_NAME=ohlc
COVER_PROFILE=coverage.out

# Build the application
build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BIN_NAME) ./cmd/ohlc

# Run the application
run:
	$(GO) run ./cmd/ohlc

# Clean build artifacts
clean:
	rm -rf $(BIN_DIR)

# Generate protobuf code
proto:
	$(PROTOC) --go_out=internal/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=internal/proto \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/ohlc.proto

# Run tests
test:
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=$(COVER_PROFILE) ./...
	$(GO) tool cover -html=$(COVER_PROFILE)

# Default target
all: proto build test

# Run development environment using docker-compose
run_dev:
	docker-compose up --build

integration-test:
	$(GO) test -v ./internal/service