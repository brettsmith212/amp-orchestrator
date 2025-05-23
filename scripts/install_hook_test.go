package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstallHook(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "hook-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "repo.git")
	cmd := exec.Command("git", "init", "--bare", repoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create bare repository: %v", err)
	}

	// Create a temporary CI script
	ciScriptPath := filepath.Join(tmpDir, "ci.sh")
	ciScriptContent := "#!/bin/bash\necho \"CI Running\"\nexit 0"
	if err := os.WriteFile(ciScriptPath, []byte(ciScriptContent), 0755); err != nil {
		t.Fatalf("Failed to create CI script: %v", err)
	}

	// Run the hook installer
	cmd = exec.Command("go", "run", "install_hook.go", "--repo", repoPath, "--ci-script", ciScriptPath)
	cmd.Env = os.Environ()
	// Set -race flag for concurrent operation safety testing
	cmd.Env = append(cmd.Env, "GORACE=halt_on_error=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Hook installation failed: %v\nOutput: %s", err, output)
	}

	// Check if the hook was created
	hookPath := filepath.Join(repoPath, "hooks", "post-receive")
	hookContent, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("Failed to read hook file: %v", err)
	}

	// Verify the hook content
	hookContentStr := string(hookContent)
	if !strings.Contains(hookContentStr, "CI_SCRIPT=\"") {
		t.Errorf("Hook does not contain CI_SCRIPT variable")
	}

	if !strings.Contains(hookContentStr, ciScriptPath) {
		t.Errorf("Hook does not reference the correct CI script path")
	}

	// Check that the ci-status directory was created
	statusDir := filepath.Join(repoPath, "ci-status")
	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		t.Errorf("ci-status directory was not created")
	}

	// Test with a non-existent repository
	cmd = exec.Command("go", "run", "install_hook.go", "--repo", filepath.Join(tmpDir, "nonexistent"))
	output, _ = cmd.CombinedOutput()
	if !strings.Contains(string(output), "Error accessing repository") {
		t.Errorf("Expected error for non-existent repository, got: %s", output)
	}
}

func TestHookParameterForwarding(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "hook-param-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a bare repository
	repoPath := filepath.Join(tmpDir, "repo.git")
	cmd := exec.Command("git", "init", "--bare", repoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create bare repository: %v", err)
	}

	// Create a mock CI script that logs its parameters
	ciScriptPath := filepath.Join(tmpDir, "ci.sh")
	logPath := filepath.Join(tmpDir, "ci.log")
	ciScriptContent := fmt.Sprintf(`#!/bin/bash
# Mock CI script that logs parameters and creates status JSON
echo "REPO_DIR: $1" >> %s
echo "REF_NAME: $2" >> %s
echo "COMMIT_HASH: $3" >> %s

# Create status directory
mkdir -p "$1/ci-status"

# Create a mock status JSON file
cat > "$1/ci-status/$3.json" << EOF
{
  "ref": "$2",
  "commit": "$3",
  "status": "PASS",
  "timestamp": "$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)",
  "output": "Mock test output"
}
EOF
`, logPath, logPath, logPath)

	if err := os.WriteFile(ciScriptPath, []byte(ciScriptContent), 0755); err != nil {
		t.Fatalf("Failed to create CI script: %v", err)
	}

	// Install the hook
	cmd = exec.Command("go", "run", "install_hook.go", "--repo", repoPath, "--ci-script", ciScriptPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Hook installation failed: %v", err)
	}

	// Create a working copy and push a commit to trigger the hook
	workingCopyPath := filepath.Join(tmpDir, "working-copy")
	cmd = exec.Command("git", "clone", repoPath, workingCopyPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to clone repository: %v", err)
	}

	// Set up git config in working copy
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workingCopyPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set git user.name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workingCopyPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to set git user.email: %v", err)
	}

	// Create and push a commit
	testFile := filepath.Join(workingCopyPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = workingCopyPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Test commit")
	cmd.Dir = workingCopyPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Get the commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = workingCopyPath
	commitHashBytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(string(commitHashBytes))

	// Push to trigger the hook
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = workingCopyPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to push: %v", err)
	}

	// Give the hook time to run
	time.Sleep(100 * time.Millisecond)

	// Verify the parameters were forwarded correctly
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read CI log: %v", err)
	}

	logStr := string(logContent)
	expectedRepoDir := fmt.Sprintf("REPO_DIR: %s", repoPath)
	expectedRefName := "REF_NAME: refs/heads/main"
	expectedCommitHash := fmt.Sprintf("COMMIT_HASH: %s", commitHash)

	if !strings.Contains(logStr, expectedRepoDir) {
		t.Errorf("Expected repository directory in log, got: %s", logStr)
	}
	if !strings.Contains(logStr, expectedRefName) {
		t.Errorf("Expected ref name in log, got: %s", logStr)
	}
	if !strings.Contains(logStr, expectedCommitHash) {
		t.Errorf("Expected commit hash in log, got: %s", logStr)
	}

	// Verify status directory was created by the hook
	statusDir := filepath.Join(repoPath, "ci-status")
	if _, err := os.Stat(statusDir); os.IsNotExist(err) {
		t.Errorf("ci-status directory was not created by hook")
	}

	// Verify status JSON file was created
	statusFile := filepath.Join(statusDir, commitHash+".json")
	statusContent, err := os.ReadFile(statusFile)
	if err != nil {
		t.Fatalf("Failed to read status file: %v", err)
	}

	// Parse and verify status JSON
	var status map[string]interface{}
	if err := json.Unmarshal(statusContent, &status); err != nil {
		t.Fatalf("Failed to parse status JSON: %v", err)
	}

	if status["ref"] != "refs/heads/main" {
		t.Errorf("Expected ref 'refs/heads/main', got %v", status["ref"])
	}
	if status["commit"] != commitHash {
		t.Errorf("Expected commit %s, got %v", commitHash, status["commit"])
	}
	if status["status"] != "PASS" {
		t.Errorf("Expected status 'PASS', got %v", status["status"])
	}
}