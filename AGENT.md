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
# Full Demo Process with AI Integration
make build
./bin/orchestrator init my-project              # Initialize new project
cd my-project
./bin/orchestrator-daemon &                     # Start daemon
./bin/orchestrator enqueue examples/hello-world.yaml  # Enqueue ticket
git --git-dir repo.git branch -a                # See agent branches
git clone repo.git project && cd project        # Clone to see code
git checkout agent-X/feat-ticket-id             # See AI-generated code

# Legacy process (manual setup)
cp config.sample.yaml config.yaml
./bin/orchestrator validate examples/avatar.yaml
./bin/orchestrator enqueue examples/avatar.yaml
```

## Project Architecture

### Core Components
- **Daemon** (`cmd/daemon`): Main orchestrator process with automatic git hook installation
- **CLI** (`cmd/cli`): Command-line tool for ticket management
- **Workers** (`internal/worker`): Agents that process tickets with real CI integration
- **Queue** (`internal/queue`): Thread-safe priority queue
- **Watcher** (`internal/watch`): File system monitoring
- **Git Utils** (`pkg/gitutils`): Git operations and worktree management
- **CI Integration** (`internal/ci`): Real CI status reading and processing

### Key Patterns
- **Bare Repository**: `repo.git/` contains only git metadata
- **Worktrees**: Temporary isolated workspaces in `tmp/agent-X/ticket-id/`
- **Processed Files**: Tickets moved to `backlog/processed/` to prevent duplicates
- **Branch Naming**: `agent-X/ticket-id` pattern for isolation
- **Priority Queue**: Lower numbers = higher priority, FIFO within same priority

### Directory Structure
```
repo.git/          # Bare git repository (metadata only)
├── hooks/         # Git hooks (post-receive for CI triggering)
├── ci-status/     # CI result JSON files (<commit-hash>.json)
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
  ci/              # CI status reading and parsing
  errors.go        # Common error types
pkg/               # Reusable packages
  gitutils/        # Git operations
scripts/           # Utility scripts
  install_hook.go  # Git hook installer
examples/          # Sample ticket files
docs/              # Documentation
ci.sh              # CI execution script
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
- Use `SkipCI: true` flag in worker tests to avoid CI timeouts
- Use `SkipAmp: true` flag in worker tests to avoid hanging on amp CLI calls

### CI Integration
- **Direct CI triggering**: Workers call `ci.sh` directly (more reliable than git hooks)
- **Status polling**: Workers poll `repo.git/ci-status/<commit-hash>.json` files  
- **Timeout handling**: 30-second timeout with 1-second polling interval
- **Error handling**: CI failures stop worker processing and clean up branches

## Important Implementation Details

### Sprint 1 Fixes Applied
1. **Duplicate Processing**: Fixed by moving processed tickets to `backlog/processed/`
2. **Git Remote Paths**: Fixed by using absolute paths before changing directories
3. **Repository Initialization**: Daemon automatically creates bare repo and initial commit
4. **Real CI Integration**: Replaced mock CI with actual test execution and status monitoring
5. **Git Hook Installation**: Daemon automatically installs post-receive hooks for CI triggering

### Worker Behavior
- Workers poll queue every 2 seconds
- Only one worker processes each ticket
- **Real AI Integration**: Workers use Amp CLI to generate actual functional applications
- **Real CI integration**: Workers trigger `ci.sh` directly after pushing code
- Workers wait for CI results (30s timeout, 1s polling) before proceeding
- Automatic cleanup of worktrees after completion

### Ticket Processing Flow
1. File added to `backlog/` directory
2. Watcher detects file and loads ticket
3. Ticket enqueued and file moved to `backlog/processed/`
4. Worker pops ticket from queue
5. Worker creates worktree and branch `agent-X/ticket-id`
6. Worker simulates work and commits changes
7. Worker triggers CI (real) and waits for results
8. Worker cleans up on completion or CI failure

### Demo Verification Points
- Each ticket processed by exactly one worker
- Branches visible with `git --git-dir repo.git branch -a`
- Actual code visible by cloning and checking out agent branch
- Processed files moved to `backlog/processed/`
- Workers return to idle state after completion
- CI status files created in `repo.git/ci-status/` with test results

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

### Prerequisites
- **Amp CLI**: Required for AI code generation (`amp --help` should work)
- **API Key**: Amp CLI must be configured with valid API credentials

## Next Sprint Preparations

### Sprint 1 Complete ✅
- ✅ **Real CI integration** (completed - replaces mock)
- ✅ **Git utility enhancements** (GetBranchCommit, hook installation)
- ✅ **Worker CI integration** (direct CI triggering and status monitoring)
- ✅ **Real AI Integration** (completed - workers use Amp CLI for actual code generation)
- ✅ **Init Command** (completed - automatic project setup with `orchestrator init`)

### Sprint 2 Goals (Updated)
- TUI interface for real-time monitoring
- Enhanced error recovery for failed worker operations
- Performance optimizations for larger codebases

### Technical Debt
- Lock mechanism for file conflicts (planned for later sprints)
- Hook triggering reliability (currently using direct CI calls as workaround)

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
- Follow `docs/DEMO.md` step-by-step
- Verify single-worker-per-ticket behavior
- Check branch creation and code generation
- Confirm processed file movement
- Verify CI integration (status files in `repo.git/ci-status/`)