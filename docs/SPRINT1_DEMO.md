# Sprint 1 Demo Guide

This document provides a step-by-step guide for demonstrating the Sprint 1 functionality of the Amp Orchestrator.

## Prerequisites

1. Go 1.24+ installed
2. Git installed and configured
3. Config file set up (copy `config.sample.yaml` to `config.yaml`)

## Demo Setup

### 1. Build the Project

```bash
make build
```

This creates two binaries:
- `./bin/orchestrator-daemon` - The main orchestrator daemon
- `./bin/orchestrator` - CLI tool for managing tickets

### 2. Initialize the Repository

The orchestrator needs a bare git repository to work with:

```bash
# Create a bare repository (optional - daemon will create if it doesn't exist)
mkdir -p repo.git
git init --bare repo.git

# The daemon will automatically create the initial commit when it starts
```

### 3. Ensure Directories Exist

```bash
# Create required directories  
mkdir -p backlog tmp ci-status metrics
# Note: A 'backlog/processed/' directory will be created automatically
```

## Demo Flow

### Step 1: Start the Daemon

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

In terminal 2, validate the example ticket:

```bash
./bin/orchestrator validate examples/avatar.yaml
```

Expected output:
```
✅ Ticket validation passed
   ID: feat-avatar-123
   Title: Add user avatar support
   Priority: 2
   Locks: [user-profile upload-system image-processing]
   Dependencies: [feat-user-auth-100 feat-file-storage-101]
```

### Step 3: Enqueue the Ticket

Enqueue the ticket for processing:

```bash
./bin/orchestrator enqueue examples/avatar.yaml
```

Expected output:
```
✅ Enqueued ticket feat-avatar-123
   File: backlog/avatar.yaml
   Title: Add user avatar support
   Priority: 2
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
Worker 2 committed changes: a4ee4666556c78055a9182d43555fd6943d478b5
Worker 2 triggering CI for branch agent-2/feat-avatar-123
Worker 2: CI triggered successfully for agent-2/feat-avatar-123
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
# Output: README.md, feature-feat-avatar-123.md

# Look at the actual code the agent wrote
cat feature-feat-avatar-123.md

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
├── README.md      # Files from current branch
└── feature-*.md   # Agent-created files (when on agent branch)
```

**Workflow**:
1. Agent creates worktree in `tmp/agent-X/ticket-id/`
2. Agent writes code and commits to branch `agent-X/ticket-id`
3. Branch is pushed to `repo.git`
4. Worktree is cleaned up from `tmp/`
5. To see changes: clone `repo.git` → `project/` and checkout agent branch

## Advanced Demo Features

### Multiple Tickets

Create additional tickets and enqueue them to see multiple workers in action:

```bash
# Create a second ticket file
cat > test-ticket-2.yaml << 'EOF'
id: "feat-test-456"
title: "Test feature"
description: "A simple test feature"
priority: 1
EOF

# Enqueue it using the CLI
./bin/orchestrator enqueue test-ticket-2.yaml

# Watch the daemon logs to see which worker picks it up
# Each ticket will be processed by exactly ONE worker
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
Worker 3: processing feat-test-456 (Test feature)
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

## What's Demonstrated

This demo shows:

1. **Ticket Management**: YAML-based ticket definition and validation
2. **File Watching**: Automatic detection of new tickets in the backlog
3. **Priority Queue**: Tickets processed in priority order
4. **Worker Orchestration**: Multiple workers processing tickets in parallel
5. **Git Integration**: Automatic branch creation and worktree management
6. **CI Integration**: Mock CI triggering (foundation for real CI)
7. **Status Monitoring**: Real-time visibility into worker activity

## Next Steps

Sprint 1 provides the foundation for:
- Real CI integration (Sprint 2)
- TUI interface (Sprint 3) 
- Lock management (Sprint 4)
- Code review automation (Sprint 5)
- Merge automation (Sprint 6)

The core orchestration loop is now functional and ready for enhancement.