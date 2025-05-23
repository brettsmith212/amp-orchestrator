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
	go vet ./...
	@echo "Lint check completed"

clean:
	rm -rf $(BIN_DIR)