# Agent Memory - Amp Orchestrator

This file contains important project information for future development sessions.

## Frequently Used Commands

### Build & Test
```bash
make build        # Builds both orchestrator-daemon and orchestrator CLI
make test         # Runs all tests
make lint         # Runs go vet on all packages
make run          # Starts the daemon (uses go run ./cmd/daemon)
make clean        # Removes bin/ directory

go test ./...                    # Run all tests
go test ./internal/package -v    # Run specific package tests with verbose output
go test ./pkg/gitutils -run TestAddWorktree  # Run specific test
```

### Demo & Manual Testing
```bash
# Sprint 1 Demo Process
cp config.sample.yaml config.yaml
make build
./bin/orchestrator-daemon &                    # Start daemon
./bin/orchestrator validate examples/avatar.yaml  # Validate ticket
./bin/orchestrator enqueue examples/avatar.yaml   # Enqueue ticket
git --git-dir repo.git branch -a                  # See agent branches
git clone repo.git project && cd project          # Clone to see code
git checkout agent-X/feat-ticket-id               # See agent's work
```

## Project Architecture

### Core Components
- **Daemon** (`cmd/daemon`): Main orchestrator process
- **CLI** (`cmd/cli`): Command-line tool for ticket management
- **Workers** (`internal/worker`): Agents that process tickets
- **Queue** (`internal/queue`): Thread-safe priority queue
- **Watcher** (`internal/watch`): File system monitoring
- **Git Utils** (`pkg/gitutils`): Git operations and worktree management

### Key Patterns
- **Bare Repository**: `repo.git/` contains only git metadata
- **Worktrees**: Temporary isolated workspaces in `tmp/agent-X/ticket-id/`
- **Processed Files**: Tickets moved to `backlog/processed/` to prevent duplicates
- **Branch Naming**: `agent-X/ticket-id` pattern for isolation
- **Priority Queue**: Lower numbers = higher priority, FIFO within same priority

### Directory Structure
```
repo.git/          # Bare git repository (metadata only)
tmp/               # Temporary worktrees (cleaned up after use)
backlog/           # New tickets (watched by daemon)
  processed/       # Processed tickets (moved here automatically)
cmd/               # Binary entry points
  daemon/          # Main orchestrator daemon
  cli/             # Command-line interface
internal/          # Internal packages
  config/          # Configuration management
  queue/           # Priority queue implementation
  ticket/          # Ticket data structures and validation
  watch/           # File system watching
  worker/          # Agent worker implementation
  errors.go        # Common error types
pkg/               # Reusable packages
  gitutils/        # Git operations
examples/          # Sample ticket files
docs/              # Documentation
```

## Code Style & Conventions

### Error Handling
- Use `internal/errors.go` for git-related errors
- Wrap errors with `internal.NewGitError(operation, path, err)`
- Always provide context in error messages

### Git Operations
- **Always use absolute paths** for git remotes (worktrees resolve paths differently)
- Create worktrees for isolated agent work
- Push to origin after commits to ensure persistence
- Clean up worktrees when work is complete

### Concurrency
- All queue operations are thread-safe
- Workers run in separate goroutines
- Use context for graceful shutdown
- Protect shared state with mutexes

### Testing
- Comprehensive test coverage for all packages
- Use temporary directories for git tests
- Mock external dependencies where appropriate
- Test both success and error paths

## Important Implementation Details

### Sprint 1 Fixes Applied
1. **Duplicate Processing**: Fixed by moving processed tickets to `backlog/processed/`
2. **Git Remote Paths**: Fixed by using absolute paths before changing directories
3. **Repository Initialization**: Daemon automatically creates bare repo and initial commit

### Worker Behavior
- Workers poll queue every 2 seconds
- Only one worker processes each ticket
- Work simulation creates feature documentation files
- CI triggering is currently mocked
- Automatic cleanup of worktrees after completion

### Ticket Processing Flow
1. File added to `backlog/` directory
2. Watcher detects file and loads ticket
3. Ticket enqueued and file moved to `backlog/processed/`
4. Worker pops ticket from queue
5. Worker creates worktree and branch `agent-X/ticket-id`
6. Worker simulates work and commits changes
7. Worker triggers CI (mock) and cleans up

### Demo Verification Points
- Each ticket processed by exactly one worker
- Branches visible with `git --git-dir repo.git branch -a`
- Actual code visible by cloning and checking out agent branch
- Processed files moved to `backlog/processed/`
- Workers return to idle state after completion

## Configuration

### Default Settings
- 3 agents by default (configurable in config.yaml)
- 5-second poll interval for file watching
- 30-second status reporting interval
- Repository: `./repo.git`, Working dir: `./tmp`
- Backlog: `./backlog`, CI status: `./ci-status`

### Required Git Configuration
```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

## Next Sprint Preparations

### Sprint 2 Goals (from implementation.md)
- Real CI integration (replace mock)
- Git utility enhancements
- Amp-Worker stub improvements

### Technical Debt
- Worker simulation should create actual code files (not just markdown)
- CI integration needs real pipeline triggers
- Error recovery for failed worker operations
- Lock mechanism for file conflicts (planned for later sprints)

## Testing Strategy

### Unit Tests
- All internal packages have comprehensive test coverage
- Git operations tested with temporary repositories
- Queue operations tested for thread safety
- Ticket validation tested for all edge cases

### Integration Tests
- End-to-end worker processing
- File watcher event handling
- Git worktree lifecycle
- CLI command validation

### Manual Verification
- Follow `docs/SPRINT1_DEMO.md` step-by-step
- Verify single-worker-per-ticket behavior
- Check branch creation and code generation
- Confirm processed file movement