package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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