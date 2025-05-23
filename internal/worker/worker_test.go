package worker

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/queue"
	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
	"github.com/brettsmith212/amp-orchestrator/pkg/gitutils"
)

func TestWorkerCreatesBranch(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	
	// Create bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := gitutils.InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	repo := gitutils.NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Create queue and add a test ticket
	q := queue.New()
	testTicket := &ticket.Ticket{
		ID:          "feat-123",
		Title:       "Test feature",
		Description: "A test feature for worker testing",
		Priority:    1,
		EstimateMin: 60,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	q.Push(testTicket)
	
	// Create worker
	config := Config{
		ID:       1,
		RepoPath: repoPath,
		WorkDir:  filepath.Join(tmpDir, "work"),
	}
	worker := New(config, q)
	
	// Start worker in background
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()
	
	// Wait for worker to process the ticket
	time.Sleep(5 * time.Second)
	
	// Cancel context to stop worker
	cancel()
	<-done
	
	// Verify that the branch was created
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	
	expectedBranch := "agent-1/feat-123"
	branchFound := false
	for _, branch := range branches {
		if strings.Contains(branch, expectedBranch) {
			branchFound = true
			break
		}
	}
	
	if !branchFound {
		t.Errorf("Expected branch %s not found. Branches: %v", expectedBranch, branches)
	}
	
	// Verify queue is empty (ticket was processed)
	if q.Len() != 0 {
		t.Errorf("Expected queue to be empty after processing, got %d tickets", q.Len())
	}
}

func TestWorkerProcessesMultipleTickets(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	
	// Create bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := gitutils.InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	repo := gitutils.NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Create queue and add multiple test tickets
	q := queue.New()
	tickets := []*ticket.Ticket{
		{
			ID:          "feat-456",
			Title:       "First feature",
			Description: "First test feature",
			Priority:    1,
			EstimateMin: 30,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "feat-789",
			Title:       "Second feature",
			Description: "Second test feature", 
			Priority:    2,
			EstimateMin: 45,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	
	for _, ticket := range tickets {
		q.Push(ticket)
	}
	
	// Create worker
	config := Config{
		ID:       2,
		RepoPath: repoPath,
		WorkDir:  filepath.Join(tmpDir, "work"),
	}
	worker := New(config, q)
	
	// Start worker in background
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()
	
	// Wait for worker to process both tickets
	time.Sleep(10 * time.Second)
	
	// Cancel context to stop worker
	cancel()
	<-done
	
	// Verify both branches were created
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	
	expectedBranches := []string{"agent-2/feat-456", "agent-2/feat-789"}
	for _, expectedBranch := range expectedBranches {
		branchFound := false
		for _, branch := range branches {
			if strings.Contains(branch, expectedBranch) {
				branchFound = true
				break
			}
		}
		if !branchFound {
			t.Errorf("Expected branch %s not found. Branches: %v", expectedBranch, branches)
		}
	}
	
	// Verify queue is empty
	if q.Len() != 0 {
		t.Errorf("Expected queue to be empty after processing, got %d tickets", q.Len())
	}
}

func TestWorkerStatus(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	
	// Create bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := gitutils.InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	repo := gitutils.NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Create empty queue and worker
	q := queue.New()
	config := Config{
		ID:       3,
		RepoPath: repoPath,
		WorkDir:  filepath.Join(tmpDir, "work"),
	}
	worker := New(config, q)
	
	// Test initial status
	status := worker.GetStatus()
	if status.ID != 3 {
		t.Errorf("Expected worker ID 3, got %d", status.ID)
	}
	if status.IsRunning {
		t.Error("Expected worker to not be running initially")
	}
	if status.CurrentTicket != nil {
		t.Error("Expected no current ticket initially")
	}
	
	// Start worker briefly to test running status
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()
	
	// Give worker time to start
	time.Sleep(100 * time.Millisecond)
	
	// Check running status
	status = worker.GetStatus()
	if !status.IsRunning {
		t.Error("Expected worker to be running")
	}
	
	// Stop worker
	cancel()
	<-done
}

func TestWorkerCITrigger(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	
	// Create bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := gitutils.InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	repo := gitutils.NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Create queue with a ticket
	q := queue.New()
	testTicket := &ticket.Ticket{
		ID:          "feat-ci-test",
		Title:       "CI test feature",
		Description: "Feature to test CI triggering",
		Priority:    1,
		EstimateMin: 30,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	q.Push(testTicket)
	
	// Create worker
	config := Config{
		ID:       4,
		RepoPath: repoPath,
		WorkDir:  filepath.Join(tmpDir, "work"),
	}
	worker := New(config, q)
	
	// Start worker in background
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()
	
	// Wait for processing
	time.Sleep(5 * time.Second)
	
	// Cancel and wait for completion
	cancel()
	<-done
	
	// Verify the branch exists (indicating CI was triggered)
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	
	expectedBranch := "agent-4/feat-ci-test"
	branchFound := false
	for _, branch := range branches {
		if strings.Contains(branch, expectedBranch) {
			branchFound = true
			break
		}
	}
	
	if !branchFound {
		t.Errorf("Expected branch %s not found, CI may not have been triggered properly", expectedBranch)
	}
}

func TestWorkerWithEmptyQueue(t *testing.T) {
	// Create test environment
	tmpDir := t.TempDir()
	
	// Create bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := gitutils.InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	repo := gitutils.NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Create empty queue
	q := queue.New()
	
	// Create worker
	config := Config{
		ID:       5,
		RepoPath: repoPath,
		WorkDir:  filepath.Join(tmpDir, "work"),
	}
	worker := New(config, q)
	
	// Start worker briefly
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()
	
	// Wait and stop
	time.Sleep(2 * time.Second)
	cancel()
	<-done
	
	// Verify no branches were created (except main/master)
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	
	// Should only have main/master branch
	agentBranches := 0
	for _, branch := range branches {
		if strings.Contains(branch, "agent-") {
			agentBranches++
		}
	}
	
	if agentBranches != 0 {
		t.Errorf("Expected no agent branches with empty queue, got %d", agentBranches)
	}
}

func TestBranchNaming(t *testing.T) {
	// Test that branch names follow the expected pattern
	tmpDir := t.TempDir()
	
	// Create bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := gitutils.InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	repo := gitutils.NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Create queue with ticket that has complex ID
	q := queue.New()
	testTicket := &ticket.Ticket{
		ID:          "feat-complex-feature-name",
		Title:       "Complex Feature",
		Description: "A feature with a complex name",
		Priority:    1,
		EstimateMin: 30,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	q.Push(testTicket)
	
	// Create worker with specific ID
	config := Config{
		ID:       42,
		RepoPath: repoPath,
		WorkDir:  filepath.Join(tmpDir, "work"),
	}
	worker := New(config, q)
	
	// Process ticket
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()
	
	time.Sleep(3 * time.Second)
	cancel()
	<-done
	
	// Check that branch follows agent-X/feat-id pattern
	cmd := exec.Command("git", "--git-dir", repoPath, "branch", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	
	branchOutput := string(output)
	expectedPattern := "agent-42/feat-complex-feature-name"
	
	if !strings.Contains(branchOutput, expectedPattern) {
		t.Errorf("Expected branch pattern %s not found in output: %s", expectedPattern, branchOutput)
	}
}