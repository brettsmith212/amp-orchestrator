package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
	"github.com/brettsmith212/amp-orchestrator/pkg/gitutils"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "init":
		var projectName string
		if len(os.Args) > 2 {
			projectName = os.Args[2]
		}
		initProject(projectName)
		
	case "validate":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s validate <ticket-file.yaml>\n", os.Args[0])
			os.Exit(1)
		}
		validateTicket(os.Args[2])
		
	case "enqueue":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s enqueue <ticket-file.yaml>\n", os.Args[0])
			os.Exit(1)
		}
		enqueueTicket(os.Args[2])
		
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [args]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  init [name]      Initialize a new orchestrator project\n")
	fmt.Fprintf(os.Stderr, "  validate <file>  Validate a ticket YAML file\n")
	fmt.Fprintf(os.Stderr, "  enqueue <file>   Enqueue a ticket by copying it to the backlog directory\n")
}

func validateTicket(filePath string) {
	// Load and validate the ticket
	t, err := ticket.Load(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Ticket validation passed\n")
	fmt.Printf("   ID: %s\n", t.ID)
	fmt.Printf("   Title: %s\n", t.Title)
	fmt.Printf("   Priority: %d\n", t.Priority)
	if len(t.Locks) > 0 {
		fmt.Printf("   Locks: %v\n", t.Locks)
	}
	if len(t.Dependencies) > 0 {
		fmt.Printf("   Dependencies: %v\n", t.Dependencies)
	}
}

func enqueueTicket(filePath string) {
	// First validate the ticket
	t, err := ticket.Load(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load ticket: %v\n", err)
		os.Exit(1)
	}
	
	// Determine backlog directory
	// Default to ./backlog, but could be made configurable
	backlogDir := "./backlog"
	if envDir := os.Getenv("ORCHESTRATOR_BACKLOG_PATH"); envDir != "" {
		backlogDir = envDir
	}
	
	// Create backlog directory if it doesn't exist
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create backlog directory: %v\n", err)
		os.Exit(1)
	}
	
	// Determine destination filename
	originalName := filepath.Base(filePath)
	destPath := filepath.Join(backlogDir, originalName)
	
	// Check if destination already exists and has the same ticket ID
	if _, err := os.Stat(destPath); err == nil {
		// File exists, check if it's the same ticket
		existingTicket, loadErr := ticket.Load(destPath)
		if loadErr == nil && existingTicket.ID == t.ID {
			fmt.Printf("⚠️  Ticket %s is already in the backlog\n", t.ID)
			return
		}
		
		// Different ticket with same filename, need to rename
		ext := filepath.Ext(originalName)
		base := originalName[:len(originalName)-len(ext)]
		for i := 1; ; i++ {
			newName := fmt.Sprintf("%s-%d%s", base, i, ext)
			destPath = filepath.Join(backlogDir, newName)
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
		}
	}
	
	// Read source file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read source file: %v\n", err)
		os.Exit(1)
	}
	
	// Write to backlog directory
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write to backlog: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Enqueued ticket %s\n", t.ID)
	fmt.Printf("   File: %s\n", destPath)
	fmt.Printf("   Title: %s\n", t.Title)
	fmt.Printf("   Priority: %d\n", t.Priority)
	
	log.Printf("Enqueued ticket %s: %s", t.ID, t.Title)
}

func initProject(projectName string) {
	// Get project name if not provided
	if projectName == "" {
		projectName = getProjectNameInteractive()
	}

	fmt.Printf("🚀 Initializing Amp Orchestrator project: %s\n\n", projectName)

	// Create project directory if it doesn't exist
	if err := os.MkdirAll(projectName, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create project directory: %v\n", err)
		os.Exit(1)
	}

	// Change into the project directory
	if err := os.Chdir(projectName); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to enter project directory: %v\n", err)
		os.Exit(1)
	}

	// Check if already initialized
	if isInitialized() {
		fmt.Fprintf(os.Stderr, "❌ Project directory already initialized (found config.yaml)\n")
		fmt.Fprintf(os.Stderr, "   Use --force to reinitialize (not implemented yet)\n")
		os.Exit(1)
	}

	// Check prerequisites
	fmt.Println("📋 Checking prerequisites...")
	checkPrerequisites()

	// Create directory structure
	fmt.Println("📁 Creating directory structure...")
	createDirectories()

	// Initialize git repository
	fmt.Println("🔧 Initializing git repository...")
	initGitRepo()

	// Copy/create configuration
	fmt.Println("⚙️  Setting up configuration...")
	setupConfig(projectName)

	// Copy scripts
	fmt.Println("📜 Setting up scripts...")
	copyScripts()

	// Create sample ticket
	fmt.Println("🎫 Creating sample ticket...")
	createSampleTicket(projectName)

	// Final instructions
	fmt.Printf("\n✅ Project initialized successfully!\n\n")
	printNextSteps(projectName)
}

func isInitialized() bool {
	_, err := os.Stat("config.yaml")
	return err == nil
}

func getProjectNameInteractive() string {
	// Try to get from current directory name
	cwd, err := os.Getwd()
	if err == nil {
		defaultName := filepath.Base(cwd)
		if defaultName != "" && defaultName != "." && defaultName != "/" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Project name [%s]: ", defaultName)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				return defaultName
			}
			return input
		}
	}

	// Fallback to asking
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Project name: ")
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func checkPrerequisites() {
	checks := []struct {
		name    string
		command string
		args    []string
	}{
		{"Git", "git", []string{"--version"}},
		{"Go", "go", []string{"version"}},
		{"jq", "jq", []string{"--version"}},
		{"Amp CLI", "amp", []string{"--version"}},
	}

	allGood := true
	for _, check := range checks {
		cmd := exec.Command(check.command, check.args...)
		if err := cmd.Run(); err != nil {
			fmt.Printf("   ❌ %s not found\n", check.name)
			allGood = false
		} else {
			fmt.Printf("   ✅ %s\n", check.name)
		}
	}

	if !allGood {
		fmt.Fprintf(os.Stderr, "\n❌ Missing prerequisites. Please install missing tools and try again.\n")
		os.Exit(1)
	}
}

func createDirectories() {
	dirs := []string{
		"backlog",
		"tmp", 
		"ci-status",
		"metrics",
		"scripts",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("   ✅ Created %s/\n", dir)
	}
}

func initGitRepo() {
	// Check if repo.git already exists
	if _, err := os.Stat("repo.git"); err == nil {
		fmt.Println("   ⚠️  repo.git already exists, skipping git initialization")
		return
	}

	// Initialize bare repository
	if err := gitutils.InitBareRepo("repo.git"); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize git repository: %v\n", err)
		os.Exit(1)
	}

	// Create initial commit
	repo := gitutils.NewRepo("repo.git")
	if err := repo.CreateInitialCommit(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create initial commit: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("   ✅ Initialized bare git repository")
}

func setupConfig(projectName string) {
	// Check if config.sample.yaml exists
	if _, err := os.Stat("config.sample.yaml"); err != nil {
		// Create a basic config if sample doesn't exist
		createBasicConfig(projectName)
	} else {
		// Copy from sample
		data, err := os.ReadFile("config.sample.yaml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to read config.sample.yaml: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile("config.yaml", data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create config.yaml: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("   ✅ Created config.yaml")
}

func createBasicConfig(projectName string) {
	config := `# Amp Orchestrator Configuration

# Repository Settings
repository:
  path: "./repo.git"  # Path to bare git repository
  workdir: "./tmp"    # Path to working directory for agents

# Agent Settings
agents:
  count: 3           # Number of agents to run in parallel
  timeout: 1800      # Timeout in seconds for agent tasks (30 minutes)

# Scheduler Settings
scheduler:
  poll_interval: 5   # Seconds between checking for new tickets
  backlog_path: "./backlog"  # Directory to watch for new ticket files
  stale_timeout: 900 # Seconds to wait before considering an agent stale (15 minutes)

# CI Settings
ci:
  status_path: "./ci-status"  # Path to store CI status files
  quick_tests: true   # Run quick tests for fast feedback

# IPC Settings
ipc:
  socket_path: "~/.orchestrator.sock"  # Unix socket for client communication

# Metrics Settings
metrics:
  enabled: true
  output_path: "./metrics"  # Directory to store metrics CSV files
`

	if err := os.WriteFile("config.yaml", []byte(config), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create config.yaml: %v\n", err)
		os.Exit(1)
	}
}

func copyScripts() {
	scriptsCreated := false
	
	// Try to copy scripts directory from parent
	if _, err := os.Stat("../scripts"); err == nil {
		cmd := exec.Command("cp", "-r", "../scripts/", "./")
		if err := cmd.Run(); err == nil {
			fmt.Println("   ✅ Copied scripts directory from project")
			scriptsCreated = true
		}
	}
	
	// Try to copy ci.sh from parent
	if _, err := os.Stat("../ci.sh"); err == nil {
		data, err := os.ReadFile("../ci.sh")
		if err == nil {
			if err := os.WriteFile("ci.sh", data, 0755); err == nil {
				fmt.Println("   ✅ Copied ci.sh from project")
				scriptsCreated = true
			}
		}
	}
	
	// Create basic scripts if nothing was copied
	if !scriptsCreated {
		createBasicScripts()
	}
}

func createBasicScripts() {
	// Create a basic ci.sh script
	ciScript := `#!/bin/bash

set -euo pipefail

# CI Script for Amp Orchestrator
# This script is called by workers to run tests

REPO_DIR="$1"
REF_NAME="$2"
COMMIT_HASH="$3"

echo "Running CI for $REF_NAME ($COMMIT_HASH)"

# Store the original working directory
ORIGINAL_DIR="$(pwd)"

# Create status directory if it doesn't exist  
STATUS_DIR="$ORIGINAL_DIR/ci-status"
mkdir -p "$STATUS_DIR"

# Create a temporary working directory
WORK_DIR=$(mktemp -d)
echo "Using working directory: $WORK_DIR"

# Cleanup function to run on exit
cleanup() {
  echo "Cleaning up $WORK_DIR"
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

# Clone the repository into working directory
git clone "$REPO_DIR" "$WORK_DIR/repo"
cd "$WORK_DIR/repo"

# Checkout the specific commit
git checkout "$COMMIT_HASH"

echo "Running tests..."

# Initialize status
STATUS="PASS"
OUTPUT=""

# Run Go tests if go.mod exists
if [ -f go.mod ]; then
  if ! OUTPUT=$(go test ./... 2>&1); then
    STATUS="FAIL"
  fi
else
  # No tests found
  OUTPUT="No tests to run"
fi

# Create status JSON file properly escaped
jq -n \
  --arg ref "$REF_NAME" \
  --arg commit "$COMMIT_HASH" \
  --arg status "$STATUS" \
  --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
  --arg output "$OUTPUT" \
  '{
    ref: $ref,
    commit: $commit,
    status: $status,
    timestamp: $timestamp,
    output: $output
  }' > "$STATUS_DIR/$COMMIT_HASH.json"

echo "CI completed with status: $STATUS"
echo "Status saved to $STATUS_DIR/$COMMIT_HASH.json"
`

	if err := os.WriteFile("scripts/ci.sh", []byte(ciScript), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create ci.sh script: %v\n", err)
		os.Exit(1)
	}

	// Also copy to current directory for direct access
	if err := os.WriteFile("ci.sh", []byte(ciScript), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create ci.sh: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("   ✅ Created basic CI script")
}

func createSampleTicket(projectName string) {
	sampleTicket := fmt.Sprintf(`id: "feat-hello-world-001"
title: "Create Hello World application"
description: "Build a simple Go application that prints 'Hello, %s!' to demonstrate the orchestrator setup"
priority: 1
locks:
  - "hello-world"
dependencies: []
tags:
  - "go"
  - "hello-world"
  - "demo"
`, projectName)

	if err := os.WriteFile("sample-ticket.yaml", []byte(sampleTicket), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create sample ticket: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("   ✅ Created sample-ticket.yaml")
}

func printNextSteps(projectName string) {
	fmt.Println("🎯 Next steps:")
	fmt.Printf("   1. Enter the directory:  cd %s\n", projectName)
	fmt.Println("   2. Copy orchestrator binaries to the project directory")
	fmt.Println("   3. Start the daemon:     ./orchestrator-daemon")
	fmt.Println("   4. Validate the sample:  ./orchestrator validate sample-ticket.yaml")
	fmt.Println("   5. Enqueue the sample:   ./orchestrator enqueue sample-ticket.yaml")
	fmt.Println("   6. Watch the magic! ✨")
	fmt.Println("")
	fmt.Println("📚 Learn more:")
	fmt.Println("   • Read docs/DEMO.md for detailed walkthrough")
	fmt.Println("   • Create custom tickets in YAML format")
	fmt.Println("   • Monitor worker activity in daemon logs")
}