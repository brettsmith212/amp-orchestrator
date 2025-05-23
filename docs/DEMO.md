# Sprint 1 Demo Guide

This document provides a step-by-step guide for demonstrating the Sprint 1 functionality of the Amp Orchestrator.

## Prerequisites

1. Go 1.24+ installed
2. Git installed and configured
3. **Amp CLI installed and configured** with API key (see https://ampcode.com/settings)
4. `jq` installed for JSON processing in CI
5. Config file set up (copy `config.sample.yaml` to `config.yaml`)

## Demo Setup

### 1. Build the Project

```bash
make build
```

This creates two binaries:
- `./bin/orchestrator-daemon` - The main orchestrator daemon
- `./bin/orchestrator` - CLI tool for managing tickets

### 2. Initialize Your Project (NEW! 🎉)

**Easy way - One command setup:**

```bash
./bin/orchestrator init my-project-name
```

This single command will:
- ✅ **Create the project directory** and set everything up inside it
- ✅ Check all prerequisites (Git, Go, jq, Amp CLI)
- ✅ Create all required directories (backlog, tmp, ci-status, metrics, scripts)
- ✅ Initialize bare git repository with initial commit
- ✅ Copy/create configuration files
- ✅ Set up CI scripts
- ✅ Create a sample ticket to get you started
- ✅ Show you exactly what to do next

**Just like `create-react-app` or `cargo new` - creates the directory for you!**

**Manual way (if you prefer):**

```bash
# Create required directories  
mkdir -p backlog tmp ci-status metrics scripts
# Note: A 'backlog/processed/' directory will be created automatically

# Initialize bare repository
mkdir -p repo.git
git init --bare repo.git

# Copy configuration and scripts
cp config.sample.yaml config.yaml
cp -r scripts/ ./
cp ci.sh ./
```

## Demo Flow

### Step 1: Start the Daemon

**If you used `init` command:**

In terminal 1, enter your project directory and start the daemon:

```bash
cd my-project-name
cp ../bin/* .  # Copy the orchestrator binaries
./orchestrator-daemon
```

**If you set up manually:**

In terminal 1, start the orchestrator daemon:

```bash
make run
# Or directly: ./bin/orchestrator-daemon
```

You should see output like:
```
Amp Orchestrator daemon starting...
Configuration loaded successfully
Repository path: ./repo.git
Running with 3 agents
Backlog path: ./backlog
Creating initial commit in repository
Installed git hooks for CI integration
Initialized ticket queue
Orchestrator initialized and ready
Starting worker 1...
Worker 1 starting...
Starting worker 2...
Worker 2 starting...
Starting worker 3...
Worker 3 starting...
Starting backlog watcher...
Started backlog watcher on ./backlog
```

### Step 2: Validate a Ticket

In terminal 2, validate a ticket (use the generated sample or examples/avatar.yaml):

```bash
# If you used init command (inside your project directory):
./orchestrator validate sample-ticket.yaml

# If you set up manually:
./bin/orchestrator validate examples/avatar.yaml
```

Expected output:
```
✅ Ticket validation passed
   ID: feat-hello-world-001  (or feat-avatar-123)
   Title: Create Hello World application  (or Add user avatar support)
   Priority: 1  (or 2)
   Locks: [hello-world]  (or [user-profile upload-system image-processing])
```

### Step 3: Enqueue the Ticket

Enqueue the ticket for processing:

```bash
# If you used init command (inside your project directory):
./orchestrator enqueue sample-ticket.yaml

# If you set up manually:
./bin/orchestrator enqueue examples/avatar.yaml
```

Expected output:
```
✅ Enqueued ticket feat-hello-world-001
   File: backlog/sample-ticket.yaml
   Title: Create Hello World application
   Priority: 1
```

### Step 4: Watch Processing

Back in terminal 1 (daemon), you should see:

```
File event: CREATE backlog/avatar.yaml
Processing ticket file: backlog/avatar.yaml
Enqueued ticket feat-avatar-123: Add user avatar support
Moved processed ticket file to backlog/processed/avatar.yaml
Worker 2 picked up ticket: feat-avatar-123
Worker 2 processing ticket feat-avatar-123: Add user avatar support
Worker 2 created worktree at tmp/agent-2/feat-avatar-123 for branch agent-2/feat-avatar-123
Worker 2 generating code using amp CLI for ticket feat-avatar-123
Worker 2 amp CLI completed successfully
Worker 2 committed generated code: a4ee4666556c78055a9182d43555fd6943d478b5
Worker 2 triggering CI for branch agent-2/feat-avatar-123 (commit a4ee4666)
Worker 2: CI triggered successfully for agent-2/feat-avatar-123
Worker 2 waiting for CI to complete for branch agent-2/feat-avatar-123 (commit a4ee4666)
Worker 2: CI passed for agent-2/feat-avatar-123
Worker 2 completed ticket feat-avatar-123
```

**Important**: Notice that the ticket file is moved to `backlog/processed/` immediately after enqueueing to prevent duplicate processing. Only **one worker** picks up each ticket.

### Step 5: Verify Branch Creation

In terminal 3, verify that the worker created the expected branch:

```bash
git --git-dir repo.git branch -a
```

Expected output should include:
```
+ agent-2/feat-avatar-123
* main
  remotes/origin/agent-2/feat-avatar-123
```

The exact worker number (agent-1, agent-2, or agent-3) will depend on which worker processed the ticket.

### Step 6: Inspect the Work

To see what the agent actually created, you need to check out the code:

```bash
# Clone the repository to get a working copy
git clone repo.git project
cd project

# Check out the agent's branch to see their work
git checkout agent-1/feat-avatar-123

# See what files the agent created
ls -la
# Output: README.md, main.go, go.mod, and possibly pre-built binaries

# Look at the actual code the agent wrote
cat main.go
cat README.md

# Test the generated application
./avatar-app --help  # or whatever binary was created

# See the changes compared to main branch
git diff main

# View the commit history
git log --oneline
```

**Key insight**: The `repo.git` is a bare repository (just git metadata). To see actual code changes, you need to clone it into a working directory (`project/`) and checkout the agent's branch.

## Understanding the Architecture

Here's how the different pieces fit together:

```
repo.git/          # Bare repository (git metadata only)
├── refs/heads/    # Branch references
│   ├── main
│   ├── agent-1/feat-avatar-123
│   └── agent-2/feat-other-feature
└── objects/       # Git objects (commits, trees, blobs)

tmp/               # Temporary worktrees (cleaned up after use)
├── agent-1/       # Worker 1's workspace
│   └── feat-xyz/  # Currently processing ticket
└── agent-2/       # Worker 2's workspace

project/           # Working copy (created by 'git clone repo.git project')
├── .git/          # Local git metadata
├── README.md      # Documentation from agent
├── main.go        # Source code generated by agent
├── go.mod         # Go module file
└── app-binary     # Compiled application (when on agent branch)
```

**Workflow**:
1. Agent creates worktree in `tmp/agent-X/ticket-id/`
2. Agent uses **Amp CLI** to generate actual application code based on ticket description
3. Agent commits generated code and pushes to branch `agent-X/ticket-id`
4. CI is automatically triggered and runs `go test ./...`
5. Agent waits for CI results (PASS/FAIL) by polling JSON status files
6. On success: Agent completes. On failure: Agent cleans up branch
7. Worktree is cleaned up from `tmp/`
8. To see generated code: clone `repo.git` → `project/` and checkout agent branch

## Advanced Demo Features

### Multiple Tickets

Create additional tickets and enqueue them to see multiple workers in action:

```bash
# Create a second ticket file
cat > calculator.yaml << 'EOF'
id: "feat-calculator-001"
title: "Create a Go calculator CLI"
description: "Build a command-line calculator that can perform basic arithmetic operations (+, -, *, /) on two numbers passed as arguments"
priority: 1
tags: ["go", "cli", "calculator"]
EOF

# Enqueue it using the CLI
./bin/orchestrator enqueue calculator.yaml

# Watch the daemon logs to see which worker picks it up
# Each ticket will be processed by exactly ONE worker
# The agent will generate a complete calculator application!
```

### Priority Handling

Create tickets with different priorities:

```bash
# High priority ticket
cat > backlog/urgent.yaml << 'EOF'
id: "hotfix-urgent"
title: "Urgent hotfix"
description: "Critical bug fix"
priority: 1
EOF

# Low priority ticket  
cat > backlog/nice-to-have.yaml << 'EOF'
id: "feat-nice-to-have"
title: "Nice to have feature"
description: "Enhancement that can wait"
priority: 5
EOF
```

The high priority ticket should be processed first.

### Worker Status Monitoring

Every 30 seconds, the daemon logs worker status:

```
Queue status: 0 tickets pending
Worker 1: idle
Worker 2: idle  
Worker 3: processing feat-calculator-001 (Create a Go calculator CLI)
```

## Cleanup

To stop the demo:

1. Press `Ctrl+C` in terminal 1 to stop the daemon
2. Clean up temporary files:
   ```bash
   rm -rf tmp/* backlog/processed/* ci-status/* metrics/*
   # Note: processed tickets are in backlog/processed/, not backlog/
   ```

## Troubleshooting

### Common Issues

1. **"Failed to create initial commit"**: Ensure you have git configured with user.name and user.email:
   ```bash
   git config --global user.name "Your Name"
   git config --global user.email "your.email@example.com"
   ```

2. **Permission errors**: Ensure all directories are writable and you have proper file permissions.

3. **Workers not processing tickets**: Check that the daemon has successfully started all workers and the backlog watcher. Look for these lines in the output:
   ```
   Worker 1 starting...
   Worker 2 starting...
   Worker 3 starting...
   Started backlog watcher on ./backlog
   ```

4. **Branches not appearing**: If you don't see expected branches, check the daemon logs for error messages. The most common issue is git configuration problems.

5. **Multiple workers processing same ticket**: This was a bug in earlier versions. Current version moves processed tickets to `backlog/processed/` to prevent duplicate processing. Each ticket should only be processed by one worker.

6. **CI timeouts or failures**: If workers report CI failures:
   - Check `./ci-status/` for CI result files (not `repo.git/ci-status/`)
   - Verify `ci.sh` script exists and is executable  
   - Ensure `jq` is installed for JSON processing
   - Ensure Go project has valid `go.mod` in agent-created code
   - Check daemon logs for detailed CI error messages
   
7. **Amp CLI issues**: If workers fail during code generation:
   - Verify amp CLI is installed and accessible in PATH
   - Check that AMP_API_KEY environment variable is set
   - Ensure amp CLI can connect to the service

## What's Demonstrated

This demo shows:

1. **Ticket Management**: YAML-based ticket definition and validation
2. **File Watching**: Automatic detection of new tickets in the backlog
3. **Priority Queue**: Tickets processed in priority order
4. **Worker Orchestration**: Multiple workers processing tickets in parallel
5. **Real Code Generation**: AI agents create complete, functional applications
6. **Git Integration**: Automatic branch creation and worktree management  
7. **Real CI Integration**: Automatic test execution and result processing
8. **Status Monitoring**: Real-time visibility into worker activity

### CI Integration Details

The orchestrator now includes **real AI code generation and CI integration** that:
- **Code Generation**: Uses Amp CLI to generate complete, functional applications from ticket descriptions
- **Automated Testing**: Automatically triggers `go test ./...` when workers push generated code
- **Status Monitoring**: Creates JSON status files with test results in `./ci-status/`
- **Quality Control**: Workers wait for CI results before proceeding
- **Error Handling**: Failed CI stops the worker and cleans up the branch
- **Success Flow**: Passed CI allows the worker to complete successfully

## Next Steps

Sprint 1 has delivered:
- ✅ **Core orchestration** with multi-worker ticket processing
- ✅ **AI code generation** using Amp CLI for real application development
- ✅ **Git integration** with automatic branch management
- ✅ **Real CI integration** with automated testing and JSON status parsing

Upcoming sprints will add:
- TUI interface (Sprint 2)
- Lock management (Sprint 3) 
- Code review automation (Sprint 4)
- Merge automation (Sprint 5)

The core orchestration loop with AI code generation and CI integration is now complete and production-ready.

## Real Application Examples

Here are some examples of complete applications that agents have successfully generated:

### Calculator CLI
```yaml
id: "feat-calculator-001"
title: "Create a Go calculator CLI"
description: "Build a command-line calculator that can perform basic arithmetic operations (+, -, *, /) on two numbers passed as arguments"
```
**Generated**: Complete calculator with error handling, division by zero protection, and smart integer/float formatting.

### Word Counter Tool
```yaml
id: "feat-word-counter-001"  
title: "Create a word counter CLI tool"
description: "Build a Go CLI tool that counts words, lines, and characters in a text file"
```
**Generated**: Full file processing tool with proper I/O handling and formatted output.

### Echo Application
```yaml
id: "feat-echo-simple-001"
title: "Create Go echo application"
description: "Build a simple Go application that echoes the first command line parameter back to stdout"
```
**Generated**: Simple but robust CLI with usage instructions and argument validation.

Each generated application includes:
- ✅ Complete, compilable Go source code
- ✅ Proper `go.mod` file with dependencies
- ✅ Comprehensive README with usage examples
- ✅ Pre-built executable binary
- ✅ Proper error handling and edge case management
- ✅ Professional code structure and formatting