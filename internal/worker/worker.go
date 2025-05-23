package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
}

// Config holds worker configuration
type Config struct {
	ID            int
	RepoPath      string
	WorkDir       string
	CIStatusDir   string
	SkipCI        bool  // For testing - skips CI wait
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
	
	// Simulate work on the ticket
	if err := w.simulateWork(t); err != nil {
		log.Printf("Worker %d failed to complete work on %s: %v", w.ID, t.ID, err)
		w.cleanup()
		return
	}
	
	// Wait for CI to complete and check results (unless skipped for testing)
	if !w.skipCI {
	 commitHash, err := w.repo.GetBranchCommit(branchName)
	if err != nil {
	 log.Printf("Worker %d failed to get commit hash for %s: %v", w.ID, t.ID, err)
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

// simulateWork creates some mock changes to simulate agent work
func (w *Worker) simulateWork(t *ticket.Ticket) error {
	// Create a feature implementation file
	featureFile := fmt.Sprintf("feature-%s.md", t.ID)
	featureContent := fmt.Sprintf(`# Feature Implementation: %s

## Description
%s

## Priority: %d

## Implementation Notes
- Created by Agent %d
- Timestamp: %s
- Estimated time: %d minutes

## Changes Made
- Generated feature skeleton
- Added basic documentation
- Ready for review

## Next Steps
- Code review
- Testing
- Deployment
`, t.Title, t.Description, t.Priority, w.ID, time.Now().Format(time.RFC3339), t.EstimateMin)
	
	featurePath := filepath.Join(w.worktreePath, featureFile)
	if err := os.WriteFile(featurePath, []byte(featureContent), 0644); err != nil {
		return fmt.Errorf("failed to write feature file: %w", err)
	}
	
	// Commit the changes
	commitMessage := fmt.Sprintf("Implement %s\n\n%s", t.Title, t.Description)
	commitHash, err := w.repo.CommitFile(w.worktreePath, featureFile, commitMessage)
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	
	log.Printf("Worker %d committed changes: %s", w.ID, commitHash)
	return nil
}

// waitForCI waits for CI to complete and checks the result
func (w *Worker) waitForCI(commitHash, branchName string) error {
	log.Printf("Worker %d waiting for CI to complete for branch %s (commit %s)", w.ID, branchName, commitHash[:8])
	
	// Use shorter timeout for testing
	maxWaitTime := 10 * time.Second
	pollInterval := 500 * time.Millisecond
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