.PHONY: build test run lint clean

BINARY_NAME=orchestrator
BIN_DIR=./bin

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/daemon

test:
	go test ./...

run:
	go run ./cmd/daemon

lint:
	# Will be updated when govet is added in step 0-2
	@echo "Lint check will be implemented in step 0-2"

clean:
	rm -rf $(BIN_DIR)