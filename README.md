# Amp Orchestrator

A lightweight, locally-hosted orchestrator that coordinates 3-5 Sourcegraph Amp coding agents so that they can deliver features faster and with higher code quality than a single diligent agent working alone.

## Purpose

This project aims to enable parallelism and peer review in AI coding agents, replicating advantages of human engineering teams while controlling merge conflicts and ensuring tests remain green.

## Getting Started

### Prerequisites

- Go 1.24+
- Git

### Installation

```bash
# Clone the repository
git clone https://github.com/brettsmith212/amp-orchestrator.git
cd amp-orchestrator

# Build the binary
make build
```

### Running

```bash
# Run the daemon
make run
```

## Development

```bash
# Run tests
make test

# Run linting (will be implemented in step 0-2)
make lint
```

## Project Structure

```
.
├── cmd/                # Command-line applications
│   ├── daemon/        # Main orchestrator daemon
│   └── cli/           # CLI interface
├── internal/          # Private application code
├── pkg/               # Public libraries
└── scripts/           # Helper scripts
```

## License

MIT