.PHONY: build test run lint clean

DAEMON_BINARY=orchestrator-daemon
CLI_BINARY=orchestrator
BIN_DIR=./bin

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(DAEMON_BINARY) ./cmd/daemon
	go build -o $(BIN_DIR)/$(CLI_BINARY) ./cmd/cli

test:
	go test ./...

run:
	go run ./cmd/daemon

lint:
	# Will be updated when govet is added in step 0-2
	@echo "Lint check will be implemented in step 0-2"

clean:
	rm -rf $(BIN_DIR)