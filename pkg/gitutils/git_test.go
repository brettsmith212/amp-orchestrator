package gitutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddWorktree(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	
	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	// Create initial commit
	repo := NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Test adding a new worktree with new branch
	worktreePath := filepath.Join(tmpDir, "worktree1")
	branchName := "agent-1/feat-test"
	
	resultPath, err := repo.AddWorktree(worktreePath, branchName)
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}
	
	// Verify the returned path matches expected
	if resultPath != worktreePath {
		t.Errorf("Expected path %s, got %s", worktreePath, resultPath)
	}
	
	// Verify the worktree directory was created
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("Worktree directory was not created")
	}
	
	// Verify the branch was created
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	
	branchFound := false
	for _, branch := range branches {
		if strings.Contains(branch, branchName) {
			branchFound = true
			break
		}
	}
	
	if !branchFound {
		t.Errorf("Branch %s was not created. Branches: %v", branchName, branches)
	}
	
	// Verify we can't create the same worktree again
	_, err = repo.AddWorktree(worktreePath, "another-branch")
	if err == nil {
		t.Error("Expected error when creating duplicate worktree")
	}
}

func TestCommitFile(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	
	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	// Create initial commit
	repo := NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Add a worktree
	worktreePath := filepath.Join(tmpDir, "worktree1")
	branchName := "agent-1/test-commit"
	
	_, err := repo.AddWorktree(worktreePath, branchName)
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}
	
	// Create a test file in the worktree
	testFileName := "test-file.txt"
	testFilePath := filepath.Join(worktreePath, testFileName)
	testContent := "This is a test file for commit testing\n"
	
	if err := os.WriteFile(testFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Get initial commit count
	initialCount, err := repo.GetCommitCount(branchName)
	if err != nil {
		t.Fatalf("Failed to get initial commit count: %v", err)
	}
	
	// Commit the file
	commitMessage := "Add test file"
	commitHash, err := repo.CommitFile(worktreePath, testFileName, commitMessage)
	if err != nil {
		t.Fatalf("CommitFile failed: %v", err)
	}
	
	// Verify commit hash is returned and looks valid
	if len(commitHash) != 40 {
		t.Errorf("Expected 40-character commit hash, got %d characters: %s", len(commitHash), commitHash)
	}
	
	// Verify commit count increased by 1
	newCount, err := repo.GetCommitCount(branchName)
	if err != nil {
		t.Fatalf("Failed to get new commit count: %v", err)
	}
	
	if newCount != initialCount+1 {
		t.Errorf("Expected commit count to increase by 1, got %d -> %d", initialCount, newCount)
	}
	
	// Verify the commit exists in the repository
	cmd := exec.Command("git", "--git-dir", repoPath, "log", "--oneline", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to check git log: %v", err)
	}
	
	logOutput := string(output)
	if !strings.Contains(logOutput, commitMessage) {
		t.Errorf("Commit message '%s' not found in log: %s", commitMessage, logOutput)
	}
}

func TestCommitFileNoChanges(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	
	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	// Create initial commit
	repo := NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Add a worktree
	worktreePath := filepath.Join(tmpDir, "worktree1")
	branchName := "agent-1/no-changes"
	
	_, err := repo.AddWorktree(worktreePath, branchName)
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}
	
	// Try to commit without any changes
	_, err = repo.CommitFile(worktreePath, "nonexistent.txt", "Should fail")
	if err == nil {
		t.Error("Expected error when committing nonexistent file")
	}
}

func TestGetCommitCount(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	
	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	// Create initial commit
	repo := NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Get commit count on main/master branch
	mainBranch := "main"
	count, err := repo.GetCommitCount(mainBranch)
	if err != nil {
		// Try master if main doesn't exist
		mainBranch = "master"
		count, err = repo.GetCommitCount(mainBranch)
		if err != nil {
			t.Fatalf("Failed to get commit count: %v", err)
		}
	}
	
	// Should have exactly 1 commit (initial commit)
	if count != 1 {
		t.Errorf("Expected 1 initial commit, got %d", count)
	}
}

func TestListBranches(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	
	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	// Create initial commit
	repo := NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// List initial branches
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	
	// Should have at least one branch (main or master)
	if len(branches) == 0 {
		t.Error("Expected at least one branch")
	}
	
	hasMainBranch := false
	for _, branch := range branches {
		if strings.Contains(branch, "main") || strings.Contains(branch, "master") {
			hasMainBranch = true
			break
		}
	}
	
	if !hasMainBranch {
		t.Errorf("Expected main or master branch in list: %v", branches)
	}
	
	// Add a worktree (creates new branch)
	worktreePath := filepath.Join(tmpDir, "worktree1")
	branchName := "feature/test-branch"
	
	_, err = repo.AddWorktree(worktreePath, branchName)
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}
	
	// List branches again
	branches, err = repo.ListBranches()
	if err != nil {
		t.Fatalf("Failed to list branches after adding worktree: %v", err)
	}
	
	// Should now include the new branch
	hasFeatureBranch := false
	for _, branch := range branches {
		if strings.Contains(branch, branchName) {
			hasFeatureBranch = true
			break
		}
	}
	
	if !hasFeatureBranch {
		t.Errorf("Expected feature branch %s in list: %v", branchName, branches)
	}
}

func TestRemoveWorktree(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	
	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "test.git")
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	// Create initial commit
	repo := NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Add a worktree
	worktreePath := filepath.Join(tmpDir, "worktree1")
	branchName := "agent-1/test-remove"
	
	_, err := repo.AddWorktree(worktreePath, branchName)
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}
	
	// Verify worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Fatal("Worktree was not created")
	}
	
	// Remove the worktree
	if err := repo.RemoveWorktree(worktreePath); err != nil {
		t.Fatalf("RemoveWorktree failed: %v", err)
	}
	
	// Verify worktree directory was removed
	if _, err := os.Stat(worktreePath); err == nil {
		t.Error("Worktree directory still exists after removal")
	}
}

func TestInitBareRepo(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "new-repo.git")
	
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("InitBareRepo failed: %v", err)
	}
	
	// Verify the repository was created
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		t.Error("Repository directory was not created")
	}
	
	// Verify it's a bare repository
	gitDir := filepath.Join(repoPath, "HEAD")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("Repository does not appear to be a git repository (no HEAD file)")
	}
	
	// Verify it's bare (no .git subdirectory)
	dotGitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(dotGitDir); err == nil {
		t.Error("Repository appears to be non-bare (has .git directory)")
	}
}

func TestIntegrationAddWorktreeAndCommit(t *testing.T) {
	// Integration test that combines AddWorktree and CommitFile
	tmpDir := t.TempDir()
	
	// Create and initialize repository
	repoPath := filepath.Join(tmpDir, "integration.git")
	if err := InitBareRepo(repoPath); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}
	
	repo := NewRepo(repoPath)
	if err := repo.CreateInitialCommit(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}
	
	// Add worktree
	worktreePath := filepath.Join(tmpDir, "agent-work")
	branchName := "agent-1/integration-test"
	
	_, err := repo.AddWorktree(worktreePath, branchName)
	if err != nil {
		t.Fatalf("AddWorktree failed: %v", err)
	}
	
	// Create and commit multiple files
	for i := 1; i <= 3; i++ {
		fileName := fmt.Sprintf("file%d.txt", i)
		filePath := filepath.Join(worktreePath, fileName)
		content := fmt.Sprintf("Content of file %d\n", i)
		
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %d: %v", i, err)
		}
		
		commitMsg := fmt.Sprintf("Add file %d", i)
		_, err := repo.CommitFile(worktreePath, fileName, commitMsg)
		if err != nil {
			t.Fatalf("Failed to commit file %d: %v", i, err)
		}
	}
	
	// Verify final commit count
	finalCount, err := repo.GetCommitCount(branchName)
	if err != nil {
		t.Fatalf("Failed to get final commit count: %v", err)
	}
	
	// Should have 4 commits total: 1 initial + 3 file commits
	expectedCount := 4
	if finalCount != expectedCount {
		t.Errorf("Expected %d commits, got %d", expectedCount, finalCount)
	}
	
	// Clean up
	if err := repo.RemoveWorktree(worktreePath); err != nil {
		t.Errorf("Failed to clean up worktree: %v", err)
	}
}