# Amp Orchestrator

A lightweight, locally-hosted orchestrator that coordinates 3-5 Sourcegraph Amp coding agents so that they can deliver features faster and with higher code quality than a single diligent agent working alone.

## Purpose

This project aims to enable parallelism and peer review in AI coding agents, replicating advantages of human engineering teams while controlling merge conflicts and ensuring tests remain green.

## Getting Started

### Prerequisites

- Go 1.24+
- Git
- **Amp CLI** with API key configured (see https://ampcode.com/settings)
- `jq` for JSON processing

### Quick Start

1. **Build the orchestrator:**
   ```bash
   git clone https://github.com/brettsmith212/amp-orchestrator.git
   cd amp-orchestrator
   make build
   ```

2. **Initialize a new project:**
   ```bash
   ./bin/orchestrator init my-ai-project
   cd my-ai-project
   cp ../bin/* .  # Copy orchestrator binaries
   ```

3. **Start the daemon:**
   ```bash
   ./orchestrator-daemon &
   ```

4. **Enqueue a ticket:**
   ```bash
   ./orchestrator validate sample-ticket.yaml
   ./orchestrator enqueue sample-ticket.yaml
   ```

5. **Watch the magic happen!** Agents will generate code automatically.

6. **See the generated code:**
   ```bash
   # See what agents created
   git --git-dir repo.git branch -a
   
   # Clone to see the code
   git clone repo.git project
   cd project
   
   # Check out agent's branch
   git branch -a  # See all branches including remotes
   git checkout agent-1/feat-hello-world-001  # Switch to agent's work
   
   # Explore the generated code
   ls -la
   cat main.go
   cat README.md
   go run .  # Test the application!
   ```

### What Just Happened?

✨ **AI agents generated a complete, functional Go application from your ticket description!**

- **Ticket** → **Generated Code** → **Tests Pass** → **Ready for Review**
- Multiple agents work in parallel on different tickets
- Each agent creates isolated branches for conflict-free development
- Real CI integration ensures code quality

## Creating Custom Tickets

Create YAML files describing what you want built:

```yaml
id: "feat-calculator-001"
title: "Create a Go calculator CLI"
description: "Build a command-line calculator that can perform basic arithmetic operations (+, -, *, /) on two numbers passed as arguments"
priority: 1
locks:
  - "calculator-module"
dependencies: []
tags:
  - "go"
  - "cli"
  - "calculator"
```

Then enqueue it:
```bash
./orchestrator validate my-ticket.yaml
./orchestrator enqueue my-ticket.yaml
```

**The agent will generate a complete calculator application with error handling, tests, and documentation!**

## Example Generated Applications

Real examples of what agents have built:

- **Calculator CLI**: Complete arithmetic operations with error handling
- **Word Counter**: File processing tool with line/word/character counting  
- **Echo App**: Simple but robust CLI with usage instructions
- **HTTP Server**: Basic web server with routing and middleware
- **Database CRUD**: Full CRUD operations with SQLite integration

Each includes:
- ✅ Complete, compilable source code
- ✅ Proper Go modules and dependencies
- ✅ Comprehensive documentation
- ✅ Error handling and validation
- ✅ Ready-to-use binaries

## Advanced Usage

```bash
# See all commands
./orchestrator --help

# Monitor worker activity
tail -f daemon.log

# Priority tickets (1 = highest priority)
priority: 1  # Will be processed first

# Lock management (prevent conflicts)
locks:
  - "user-auth"    # This ticket locks user auth system
  - "database"     # And database layer

# Dependencies (must be completed first)  
dependencies:
  - "feat-user-auth-100"
  - "feat-database-setup-101"
```

## Development

```bash
# Run tests
make test

# Run linting
make lint
```

## Architecture

```
🎫 Ticket (YAML) → 📁 Backlog → 🤖 Agent → 🧠 Amp CLI → 💻 Generated Code → 🧪 CI → ✅ Ready
```

- **Ticket Queue**: Priority-based processing with dependency management
- **Worker Agents**: 3 agents by default, process tickets in parallel  
- **Git Worktrees**: Isolated workspaces prevent merge conflicts
- **Amp CLI Integration**: Real AI code generation from ticket descriptions
- **CI Pipeline**: Automated testing ensures code quality
- **Branch Management**: Each ticket gets its own branch for review

## Documentation

- **[📖 Full Demo Guide](docs/DEMO.md)** - Complete walkthrough with examples
- **[🎯 Implementation Details](implementation.md)** - Technical architecture  
- **[📋 Agent Memory](AGENT.md)** - Development guidelines and patterns

## Project Structure

```
.
├── cmd/                    # Command-line applications
│   ├── daemon/            # Main orchestrator daemon
│   └── cli/               # CLI interface (init, validate, enqueue)
├── internal/              # Private application code
│   ├── ci/               # CI status integration
│   ├── config/           # Configuration management
│   ├── queue/            # Priority ticket queue
│   ├── ticket/           # Ticket validation & parsing
│   ├── watch/            # File system watching
│   └── worker/           # Agent worker implementation
├── pkg/                   # Public libraries
│   └── gitutils/         # Git operations & worktree management
├── scripts/               # Helper scripts
│   └── install_hook.go   # Git hook installation
├── docs/                  # Documentation
│   └── DEMO.md           # Complete walkthrough
├── examples/              # Sample tickets
├── ci.sh                 # CI execution script
└── config.sample.yaml    # Sample configuration
```

## License

MIT