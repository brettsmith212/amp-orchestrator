package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/ci"
	"github.com/brettsmith212/amp-orchestrator/internal/queue"
	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
	"github.com/brettsmith212/amp-orchestrator/pkg/gitutils"
)

// Worker represents an Amp coding agent worker
type Worker struct {
	ID             int
	repo           *gitutils.GitRepo
	workDir        string
	queue          *queue.Queue
	isRunning      bool
	currentTask    *ticket.Ticket
	worktreePath   string
	ciStatusReader *ci.StatusReader
	skipCI         bool
	skipAmp        bool
}

// Config holds worker configuration
type Config struct {
	ID          int
	RepoPath    string
	WorkDir     string
	CIStatusDir string
	SkipCI      bool // For testing - skips CI wait
	SkipAmp     bool // For testing - skips amp CLI and creates mock files
}

// New creates a new worker instance
func New(config Config, q *queue.Queue) *Worker {
	repo := gitutils.NewRepo(config.RepoPath)
	ciStatusReader := ci.NewStatusReader(config.CIStatusDir)

	return &Worker{
		ID:             config.ID,
		repo:           repo,
		workDir:        config.WorkDir,
		queue:          q,
		ciStatusReader: ciStatusReader,
		skipCI:         config.SkipCI,
		skipAmp:        config.SkipAmp,
	}
}

// Start begins the worker's main loop
func (w *Worker) Start(ctx context.Context) error {
	w.isRunning = true
	log.Printf("Worker %d starting...", w.ID)

	// Create worker's base directory
	workerDir := filepath.Join(w.workDir, fmt.Sprintf("agent-%d", w.ID))
	if err := os.MkdirAll(workerDir, 0755); err != nil {
		return fmt.Errorf("failed to create worker directory: %w", err)
	}

	// Main worker loop
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping...", w.ID)
			w.isRunning = false
			w.cleanup()
			return nil

		case <-ticker.C:
			if w.currentTask == nil {
				// Try to get a new ticket from the queue
				if ticket := w.queue.Pop(); ticket != nil {
					log.Printf("Worker %d picked up ticket: %s", w.ID, ticket.ID)
					w.processTicket(ticket)
				}
			}
		}
	}
}

// processTicket handles a ticket from start to finish
func (w *Worker) processTicket(t *ticket.Ticket) {
	w.currentTask = t

	log.Printf("Worker %d processing ticket %s: %s", w.ID, t.ID, t.Title)

	// Generate branch name
	branchName := fmt.Sprintf("agent-%d/%s", w.ID, t.ID)

	// Create worktree for this ticket
	worktreePath := filepath.Join(w.workDir, fmt.Sprintf("agent-%d", w.ID), t.ID)

	// Clean up any existing worktree first
	if w.worktreePath != "" {
		w.cleanupWorktree()
	}

	// Create new worktree
	resultPath, err := w.repo.AddWorktree(worktreePath, branchName)
	if err != nil {
		log.Printf("Worker %d failed to create worktree for %s: %v", w.ID, t.ID, err)
		w.currentTask = nil
		return
	}

	w.worktreePath = resultPath
	log.Printf("Worker %d created worktree at %s for branch %s", w.ID, resultPath, branchName)

	// Implement the feature using amp CLI
	if err := w.implementFeature(t); err != nil {
		log.Printf("Worker %d failed to complete work on %s: %v", w.ID, t.ID, err)
		w.cleanup()
		return
	}

	// Trigger CI and wait for results (unless skipped for testing)
	if !w.skipCI {
		commitHash, err := w.repo.GetBranchCommit(branchName)
		if err != nil {
			log.Printf("Worker %d failed to get commit hash for %s: %v", w.ID, t.ID, err)
			w.cleanup()
			return
		}

		// Trigger CI manually since git hooks might not be reliable from worktrees
		if err := w.triggerCI(branchName, commitHash); err != nil {
			log.Printf("Worker %d failed to trigger CI for %s: %v", w.ID, t.ID, err)
			w.cleanup()
			return
		}

		if err := w.waitForCI(commitHash, branchName); err != nil {
			log.Printf("Worker %d CI failed for %s: %v", w.ID, t.ID, err)
			w.cleanup()
			return
		}
	} else {
		log.Printf("Worker %d: CI skipped for testing", w.ID)
	}

	log.Printf("Worker %d completed ticket %s", w.ID, t.ID)

	// Mark task as complete
	w.currentTask = nil
}

// implementFeature uses the amp CLI to generate actual code for the ticket
func (w *Worker) implementFeature(t *ticket.Ticket) error {
	if w.skipAmp {
		// For testing: create mock files instead of using amp CLI
		return w.createMockImplementation(t)
	}

	// Create a detailed prompt for the amp agent
	prompt := w.createPrompt(t)

	// Use amp CLI to generate the actual implementation
	log.Printf("Worker %d generating code using amp CLI for ticket %s", w.ID, t.ID)

	cmd := exec.Command("amp", "--no-notifications")
	cmd.Dir = w.worktreePath
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Worker %d amp CLI error output: %s", w.ID, string(output))
		return fmt.Errorf("amp CLI failed: %w", err)
	}

	log.Printf("Worker %d amp CLI completed successfully", w.ID)

	// Add all generated files to git
	if err := w.addAllChanges(); err != nil {
		return fmt.Errorf("failed to add generated files: %w", err)
	}

	// Commit all the changes
	commitMessage := fmt.Sprintf("Implement %s\n\n%s\n\nGenerated by Agent %d using amp CLI", t.Title, t.Description, w.ID)
	commitHash, err := w.commitAllChanges(commitMessage)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Printf("Worker %d committed generated code: %s", w.ID, commitHash)
	return nil
}

// createPrompt generates a detailed prompt for the amp agent based on the ticket
func (w *Worker) createPrompt(t *ticket.Ticket) string {
	prompt := fmt.Sprintf(`You are an AI coding agent working on ticket %s: %s

Description: %s
Priority: %d

`, t.ID, t.Title, t.Description, t.Priority)

	// Add dependencies context if they exist
	if len(t.Dependencies) > 0 {
		prompt += "Dependencies (these should already be implemented):\n"
		for _, dep := range t.Dependencies {
			prompt += fmt.Sprintf("- %s\n", dep)
		}
		prompt += "\n"
	}

	// Add locks context if they exist
	if len(t.Locks) > 0 {
		prompt += "This ticket locks the following components (avoid conflicts):\n"
		for _, lock := range t.Locks {
			prompt += fmt.Sprintf("- %s\n", lock)
		}
		prompt += "\n"
	}

	// Add tags context if they exist
	if len(t.Tags) > 0 {
		prompt += "Tags: "
		for i, tag := range t.Tags {
			if i > 0 {
				prompt += ", "
			}
			prompt += tag
		}
		prompt += "\n\n"
	}

	prompt += `Please implement this feature completely. Create all necessary files including:
- Source code files (main.go, etc.)
- Go module file (go.mod) if needed
- README.md with usage instructions
- Any configuration files needed

Make sure the implementation is production-ready, includes proper error handling, and follows Go best practices.

Work in the current directory. Do not explain what you're doing, just implement the solution.`

	return prompt
}

// addAllChanges adds all modified and new files to git
func (w *Worker) addAllChanges() error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = w.worktreePath

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Worker %d git add error: %s", w.ID, string(output))
		return fmt.Errorf("git add failed: %w", err)
	}

	return nil
}

// commitAllChanges commits all staged changes and pushes to origin
func (w *Worker) commitAllChanges(commitMessage string) (string, error) {
	// Get absolute path to repository before changing directories
	absRepoPath, err := filepath.Abs(w.repo.Path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute repo path: %w", err)
	}

	// Change to worktree directory for git operations
	originalDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(w.worktreePath); err != nil {
		return "", fmt.Errorf("failed to change to worktree directory: %w", err)
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to check git status: %w", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return "", fmt.Errorf("no changes to commit")
	}

	// Commit the changes
	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	if output, err := commitCmd.CombinedOutput(); err != nil {
		log.Printf("Worker %d git commit error: %s", w.ID, string(output))
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	// Get the commit hash
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashOutput, err := hashCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	commitHash := strings.TrimSpace(string(hashOutput))

	// Get current branch name
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchOutput, err := branchCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	currentBranch := strings.TrimSpace(string(branchOutput))

	// Configure the remote to point to the bare repository
	remoteCmd := exec.Command("git", "remote", "add", "origin", absRepoPath)
	if _, err := remoteCmd.CombinedOutput(); err != nil {
		// Remote might already exist, try to set the URL instead
		remoteCmd = exec.Command("git", "remote", "set-url", "origin", absRepoPath)
		if output, err := remoteCmd.CombinedOutput(); err != nil {
			log.Printf("Worker %d git remote error: %s", w.ID, string(output))
			return "", fmt.Errorf("failed to configure git remote: %w", err)
		}
	}

	// Push the commit
	pushCmd := exec.Command("git", "push", "origin", currentBranch)
	if output, err := pushCmd.CombinedOutput(); err != nil {
		log.Printf("Worker %d git push error: %s", w.ID, string(output))
		return "", fmt.Errorf("git push failed: %w", err)
	}

	return commitHash, nil
}

// createMockImplementation creates mock files for testing (when skipAmp is true)
func (w *Worker) createMockImplementation(t *ticket.Ticket) error {
	// Create a simple mock main.go file
	mainGoContent := fmt.Sprintf(`package main

import "fmt"

func main() {
	fmt.Println("Hello from %s!")
	fmt.Println("Description: %s")
	fmt.Println("Generated by Agent %d for testing")
}
`, t.Title, t.Description, w.ID)

	mainGoPath := filepath.Join(w.worktreePath, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGoContent), 0644); err != nil {
		return fmt.Errorf("failed to write mock main.go: %w", err)
	}

	// Create a mock go.mod file
	goModContent := fmt.Sprintf("module test-app-%s\n\ngo 1.21\n", t.ID)
	goModPath := filepath.Join(w.worktreePath, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write mock go.mod: %w", err)
	}

	// Create a mock README.md file
	readmeContent := fmt.Sprintf(`# %s

%s

## Testing

This is a mock implementation generated for testing purposes.

Generated by Agent %d.
`, t.Title, t.Description, w.ID)

	readmePath := filepath.Join(w.worktreePath, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write mock README.md: %w", err)
	}

	log.Printf("Worker %d created mock implementation for testing", w.ID)

	// Add all generated files to git
	if err := w.addAllChanges(); err != nil {
		return fmt.Errorf("failed to add generated files: %w", err)
	}

	// Commit all the changes
	commitMessage := fmt.Sprintf("Implement %s\n\n%s\n\nMock implementation by Agent %d for testing", t.Title, t.Description, w.ID)
	commitHash, err := w.commitAllChanges(commitMessage)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	log.Printf("Worker %d committed mock implementation: %s", w.ID, commitHash)
	return nil
}

// waitForCI waits for CI to complete and checks the result
func (w *Worker) waitForCI(commitHash, branchName string) error {
	log.Printf("Worker %d waiting for CI to complete for branch %s (commit %s)", w.ID, branchName, commitHash[:8])

	// Use reasonable timeout and polling interval
	maxWaitTime := 30 * time.Second
	pollInterval := 1 * time.Second
	timeout := time.After(maxWaitTime)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for CI results after %v", maxWaitTime)

		case <-ticker.C:
			// Check if CI status exists
			if w.ciStatusReader.HasStatus(commitHash) {
				// Check if CI passed
				passing, err := w.ciStatusReader.IsPassing(commitHash)
				if err != nil {
					return fmt.Errorf("failed to check CI status: %w", err)
				}

				if passing {
					log.Printf("Worker %d: CI passed for %s", w.ID, branchName)
					return nil
				} else {
					// Get detailed status for logging
					status, err := w.ciStatusReader.GetStatus(commitHash)
					if err != nil {
						return fmt.Errorf("CI failed and unable to get details: %w", err)
					}
					return fmt.Errorf("CI failed for %s: %s", branchName, status.Output)
				}
			}
			// CI status not ready yet, continue polling
		}
	}
}

// triggerCI manually triggers the CI script for a branch and commit
func (w *Worker) triggerCI(branchName, commitHash string) error {
	log.Printf("Worker %d triggering CI for branch %s (commit %s)", w.ID, branchName, commitHash[:8])

	// Find the ci.sh script path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	// Assume ci.sh is in the project root (parent of bin/)
	projectRoot := filepath.Dir(filepath.Dir(execPath))
	ciScriptPath := filepath.Join(projectRoot, "ci.sh")

	// Check if ci.sh exists, if not use the current directory
	if _, err := os.Stat(ciScriptPath); os.IsNotExist(err) {
		// Fall back to current working directory
		ciScriptPath = "ci.sh"
	}

	// Get absolute path to repository
	repoPath, err := filepath.Abs(w.repo.Path)
	if err != nil {
		return fmt.Errorf("failed to get absolute repo path: %w", err)
	}

	// Run the CI script: ci.sh <repo_path> <ref_name> <commit_hash>
	refName := "refs/heads/" + branchName
	cmd := exec.Command(ciScriptPath, repoPath, refName, commitHash)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Worker %d CI script output: %s", w.ID, string(output))
		return fmt.Errorf("CI script failed: %w", err)
	}

	log.Printf("Worker %d: CI triggered successfully for %s", w.ID, branchName)
	return nil
}

// cleanup cleans up worker resources
func (w *Worker) cleanup() {
	if w.worktreePath != "" {
		w.cleanupWorktree()
	}
	w.currentTask = nil
}

// cleanupWorktree removes the current worktree
func (w *Worker) cleanupWorktree() {
	if w.worktreePath == "" {
		return
	}

	log.Printf("Worker %d cleaning up worktree: %s", w.ID, w.worktreePath)

	if err := w.repo.RemoveWorktree(w.worktreePath); err != nil {
		log.Printf("Worker %d failed to remove worktree %s: %v", w.ID, w.worktreePath, err)
	}

	w.worktreePath = ""
}

// GetStatus returns the current status of the worker
func (w *Worker) GetStatus() WorkerStatus {
	status := WorkerStatus{
		ID:        w.ID,
		IsRunning: w.isRunning,
	}

	if w.currentTask != nil {
		status.CurrentTicket = &TicketInfo{
			ID:    w.currentTask.ID,
			Title: w.currentTask.Title,
		}
		status.WorktreePath = w.worktreePath
	}

	return status
}

// WorkerStatus represents the current state of a worker
type WorkerStatus struct {
	ID            int         `json:"id"`
	IsRunning     bool        `json:"is_running"`
	CurrentTicket *TicketInfo `json:"current_ticket,omitempty"`
	WorktreePath  string      `json:"worktree_path,omitempty"`
}

// TicketInfo holds basic ticket information for status reporting
type TicketInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
