package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/queue"
	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
	"github.com/brettsmith212/amp-orchestrator/pkg/gitutils"
)

// Worker represents an Amp coding agent worker
type Worker struct {
	ID          int
	repo        *gitutils.GitRepo
	workDir     string
	queue       *queue.Queue
	isRunning   bool
	currentTask *ticket.Ticket
	worktreePath string
}

// Config holds worker configuration
type Config struct {
	ID       int
	RepoPath string
	WorkDir  string
}

// New creates a new worker instance
func New(config Config, q *queue.Queue) *Worker {
	repo := gitutils.NewRepo(config.RepoPath)
	
	return &Worker{
		ID:      config.ID,
		repo:    repo,
		workDir: config.WorkDir,
		queue:   q,
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
	
	// Trigger CI check (mock for now)
	if err := w.triggerCI(branchName); err != nil {
		log.Printf("Worker %d failed to trigger CI for %s: %v", w.ID, t.ID, err)
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

// triggerCI simulates triggering the CI system
func (w *Worker) triggerCI(branchName string) error {
	log.Printf("Worker %d triggering CI for branch %s", w.ID, branchName)
	
	// For now, this is just a mock implementation
	// In a real system, this would trigger the actual CI pipeline
	
	// Simulate CI processing time
	time.Sleep(100 * time.Millisecond)
	
	// Mock CI status (always pass for now)
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