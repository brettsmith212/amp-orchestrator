package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const postReceiveTemplate = `#!/bin/bash

# Post-receive hook for Amp Orchestrator
# This hook runs CI tests on received commits

CI_SCRIPT="%s"

# Create ci-status directory if it doesn't exist
mkdir -p "$(git rev-parse --git-dir)/ci-status"

# Read each ref update from stdin
while read oldrev newrev refname; do
  # Only run CI for branch updates
  if [[ $refname == refs/heads/* ]]; then
    branch=$(echo $refname | sed 's|^refs/heads/||')
    echo "Running CI for $branch..."
    
    # Get the repository path
    repo_dir=$(git rev-parse --git-dir)
    if [[ "$repo_dir" == ".git" ]]; then
      # Regular repository
      repo_dir=$(pwd)
    elif [[ "$repo_dir" == "." ]]; then
      # Bare repository
      repo_dir=$(pwd)
    else
      # Full path already
      repo_dir=$(readlink -f "$repo_dir/..")
    fi
    
    # Run the CI script
    "$CI_SCRIPT" "$repo_dir" "$refname" "$newrev"
  fi
done
`

func main() {
	// Parse command-line flags
	repoPath := flag.String("repo", "", "Path to the Git repository")
	ciScript := flag.String("ci-script", "", "Path to CI script (defaults to ../ci.sh relative to this script)")
	flag.Parse()

	// Validate repository path
	if *repoPath == "" {
		fmt.Println("Error: repository path is required")
		fmt.Println("Usage: install_hook --repo=<path-to-repo> [--ci-script=<path-to-ci-script>]")
		os.Exit(1)
	}

	// Resolve absolute path to repository
	absRepoPath, err := filepath.Abs(*repoPath)
	if err != nil {
		fmt.Printf("Error resolving repository path: %v\n", err)
		os.Exit(1)
	}

	// Check if the repository exists
	repoInfo, err := os.Stat(absRepoPath)
	if err != nil {
		fmt.Printf("Error accessing repository: %v\n", err)
		os.Exit(1)
	}

	if !repoInfo.IsDir() {
		fmt.Printf("Error: %s is not a directory\n", absRepoPath)
		os.Exit(1)
	}

	// Determine path to CI script
	ciScriptPath := *ciScript
	if ciScriptPath == "" {
		// Default to ../ci.sh relative to this script
		execPath, err := os.Executable()
		if err != nil {
			fmt.Printf("Error determining executable path: %v\n", err)
			os.Exit(1)
		}

		execDir := filepath.Dir(execPath)
		ciScriptPath = filepath.Join(filepath.Dir(execDir), "ci.sh")
	}

	// Resolve absolute path to CI script
	absCIScriptPath, err := filepath.Abs(ciScriptPath)
	if err != nil {
		fmt.Printf("Error resolving CI script path: %v\n", err)
		os.Exit(1)
	}

	// Check if the CI script exists and is executable
	ciScriptInfo, err := os.Stat(absCIScriptPath)
	if err != nil {
		fmt.Printf("Error accessing CI script: %v\n", err)
		os.Exit(1)
	}

	if ciScriptInfo.IsDir() {
		fmt.Printf("Error: %s is a directory, not a script\n", absCIScriptPath)
		os.Exit(1)
	}

	// Check if script is executable
	if ciScriptInfo.Mode()&0111 == 0 {
		fmt.Printf("Warning: CI script %s is not executable\n", absCIScriptPath)
		fmt.Println("Attempting to make it executable...")

		err = os.Chmod(absCIScriptPath, 0755)
		if err != nil {
			fmt.Printf("Error making CI script executable: %v\n", err)
			os.Exit(1)
		}
	}

	// Determine hooks directory
	hooksDir := filepath.Join(absRepoPath, "hooks")
	
	// Handle both bare and non-bare repositories
	gitDir := filepath.Join(absRepoPath, ".git")
	gitDirInfo, err := os.Stat(gitDir)
	if err == nil && gitDirInfo.IsDir() {
		// Non-bare repository
		hooksDir = filepath.Join(gitDir, "hooks")
	}

	// Create hooks directory if it doesn't exist
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		fmt.Printf("Error creating hooks directory: %v\n", err)
		os.Exit(1)
	}

	// Format post-receive hook with CI script path
	postReceiveContent := fmt.Sprintf(postReceiveTemplate, absCIScriptPath)

	// Write post-receive hook
	postReceivePath := filepath.Join(hooksDir, "post-receive")
	if err := os.WriteFile(postReceivePath, []byte(postReceiveContent), 0755); err != nil {
		fmt.Printf("Error writing post-receive hook: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully installed post-receive hook in %s\n", postReceivePath)
	fmt.Printf("Hook will use CI script at %s\n", absCIScriptPath)

	// Create ci-status directory if it doesn't exist
	statusDir := filepath.Join(absRepoPath, "ci-status")
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		fmt.Printf("Error creating ci-status directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created ci-status directory at %s\n", statusDir)
}