package ci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStatusReader_GetStatus(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Create test status file
	commitHash := "abc123def456"
	testStatus := Status{
		Ref:       "refs/heads/agent-1/feat-123",
		Commit:    commitHash,
		Status:    "PASS",
		Timestamp: time.Now().UTC(),
		Output:    "All tests passed",
	}
	
	statusFile := filepath.Join(tempDir, commitHash+".json")
	data, err := json.Marshal(testStatus)
	if err != nil {
		t.Fatalf("Failed to marshal test status: %v", err)
	}
	
	if err := os.WriteFile(statusFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test status file: %v", err)
	}
	
	// Test reading status
	reader := NewStatusReader(tempDir)
	status, err := reader.GetStatus(commitHash)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	
	if status.Commit != commitHash {
		t.Errorf("Expected commit %s, got %s", commitHash, status.Commit)
	}
	if status.Status != "PASS" {
		t.Errorf("Expected status PASS, got %s", status.Status)
	}
	if status.Ref != "refs/heads/agent-1/feat-123" {
		t.Errorf("Expected ref refs/heads/agent-1/feat-123, got %s", status.Ref)
	}
}

func TestStatusReader_GetStatus_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	reader := NewStatusReader(tempDir)
	
	_, err := reader.GetStatus("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent status, got nil")
	}
	
	// Check that error message contains the expected text
	expected := "CI status not found for commit nonexistent"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestStatusReader_HasStatus(t *testing.T) {
	tempDir := t.TempDir()
	reader := NewStatusReader(tempDir)
	
	commitHash := "test123"
	
	// Should not exist initially
	if reader.HasStatus(commitHash) {
		t.Error("HasStatus should return false for nonexistent status")
	}
	
	// Create status file
	testStatus := Status{
		Ref:    "refs/heads/test",
		Commit: commitHash,
		Status: "PASS",
	}
	
	statusFile := filepath.Join(tempDir, commitHash+".json")
	data, err := json.Marshal(testStatus)
	if err != nil {
		t.Fatalf("Failed to marshal test status: %v", err)
	}
	
	if err := os.WriteFile(statusFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test status file: %v", err)
	}
	
	// Should exist now
	if !reader.HasStatus(commitHash) {
		t.Error("HasStatus should return true for existing status")
	}
}

func TestStatusReader_IsPassing(t *testing.T) {
	tempDir := t.TempDir()
	reader := NewStatusReader(tempDir)
	
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"passing", "PASS", true},
		{"failing", "FAIL", false},
		{"unknown", "UNKNOWN", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitHash := "test-" + tt.name
			testStatus := Status{
				Ref:    "refs/heads/test",
				Commit: commitHash,
				Status: tt.status,
			}
			
			statusFile := filepath.Join(tempDir, commitHash+".json")
			data, err := json.Marshal(testStatus)
			if err != nil {
				t.Fatalf("Failed to marshal test status: %v", err)
			}
			
			if err := os.WriteFile(statusFile, data, 0644); err != nil {
				t.Fatalf("Failed to write test status file: %v", err)
			}
			
			passing, err := reader.IsPassing(commitHash)
			if err != nil {
				t.Fatalf("Failed to check if passing: %v", err)
			}
			
			if passing != tt.expected {
				t.Errorf("Expected IsPassing to return %v, got %v", tt.expected, passing)
			}
		})
	}
}

func TestStatusReader_ListStatuses(t *testing.T) {
	tempDir := t.TempDir()
	reader := NewStatusReader(tempDir)
	
	// Create multiple status files
	statuses := []Status{
		{Ref: "refs/heads/feat-1", Commit: "abc123", Status: "PASS"},
		{Ref: "refs/heads/feat-2", Commit: "def456", Status: "FAIL"},
		{Ref: "refs/heads/feat-3", Commit: "ghi789", Status: "PASS"},
	}
	
	for _, status := range statuses {
		statusFile := filepath.Join(tempDir, status.Commit+".json")
		data, err := json.Marshal(status)
		if err != nil {
			t.Fatalf("Failed to marshal status: %v", err)
		}
		
		if err := os.WriteFile(statusFile, data, 0644); err != nil {
			t.Fatalf("Failed to write status file: %v", err)
		}
	}
	
	// List all statuses
	result, err := reader.ListStatuses()
	if err != nil {
		t.Fatalf("Failed to list statuses: %v", err)
	}
	
	if len(result) != len(statuses) {
		t.Errorf("Expected %d statuses, got %d", len(statuses), len(result))
	}
	
	// Verify all expected commits are present
	commitMap := make(map[string]bool)
	for _, status := range result {
		commitMap[status.Commit] = true
	}
	
	for _, expected := range statuses {
		if !commitMap[expected.Commit] {
			t.Errorf("Expected commit %s not found in results", expected.Commit)
		}
	}
}

func TestStatusReader_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	reader := NewStatusReader(tempDir)
	
	// Create file with invalid JSON
	commitHash := "invalid"
	statusFile := filepath.Join(tempDir, commitHash+".json")
	if err := os.WriteFile(statusFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}
	
	_, err := reader.GetStatus(commitHash)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}