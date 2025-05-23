package ticket

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadValidTicket(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	
	validYAML := `id: "feat-123"
title: "Add user avatar support"
description: "Implement user avatar upload and display functionality"
priority: 2
locks:
  - "user-profile"
  - "upload-system"
dependencies:
  - "feat-100"
estimate_min: 120
tags:
  - "frontend"
  - "backend"`

	testFile := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(testFile, []byte(validYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ticket, err := Load(testFile)
	if err != nil {
		t.Fatalf("Expected valid YAML to parse, got error: %v", err)
	}

	// Verify all fields were parsed correctly
	if ticket.ID != "feat-123" {
		t.Errorf("Expected ID 'feat-123', got '%s'", ticket.ID)
	}
	
	if ticket.Title != "Add user avatar support" {
		t.Errorf("Expected title 'Add user avatar support', got '%s'", ticket.Title)
	}
	
	if ticket.Description != "Implement user avatar upload and display functionality" {
		t.Errorf("Expected specific description, got '%s'", ticket.Description)
	}
	
	if ticket.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", ticket.Priority)
	}
	
	if len(ticket.Locks) != 2 {
		t.Errorf("Expected 2 locks, got %d", len(ticket.Locks))
	}
	
	if len(ticket.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(ticket.Dependencies))
	}
	
	if ticket.EstimateMin != 120 {
		t.Errorf("Expected estimate 120, got %d", ticket.EstimateMin)
	}
	
	if len(ticket.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(ticket.Tags))
	}
	
	// Verify timestamps were set
	if ticket.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	
	if ticket.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestLoadMissingID(t *testing.T) {
	missingIDYAML := `title: "Test ticket"
description: "A test ticket without ID"
priority: 1`

	_, err := LoadFromBytes([]byte(missingIDYAML))
	if err == nil {
		t.Error("Expected error for missing ID, got nil")
	}
	
	if err.Error() != "validation failed: ticket ID is required" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoadMissingTitle(t *testing.T) {
	missingTitleYAML := `id: "test-123"
description: "A test ticket without title"
priority: 1`

	_, err := LoadFromBytes([]byte(missingTitleYAML))
	if err == nil {
		t.Error("Expected error for missing title, got nil")
	}
	
	if err.Error() != "validation failed: ticket title is required" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoadMissingDescription(t *testing.T) {
	missingDescYAML := `id: "test-123"
title: "Test ticket"
priority: 1`

	_, err := LoadFromBytes([]byte(missingDescYAML))
	if err == nil {
		t.Error("Expected error for missing description, got nil")
	}
	
	if err.Error() != "validation failed: ticket description is required" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoadInvalidPriority(t *testing.T) {
	invalidPriorityYAML := `id: "test-123"
title: "Test ticket"
description: "A test ticket with invalid priority"
priority: 10`

	_, err := LoadFromBytes([]byte(invalidPriorityYAML))
	if err == nil {
		t.Error("Expected error for invalid priority, got nil")
	}
	
	if err.Error() != "validation failed: ticket priority must be between 1 and 5" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	invalidYAML := `id: "test-123"
title: "Test ticket"
description: "Invalid YAML
priority: 1`

	_, err := LoadFromBytes([]byte(invalidYAML))
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestTicketValidate(t *testing.T) {
	// Test valid ticket
	validTicket := &Ticket{
		ID:          "test-123",
		Title:       "Test ticket",
		Description: "A test ticket",
		Priority:    3,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if err := validTicket.Validate(); err != nil {
		t.Errorf("Expected valid ticket to pass validation, got: %v", err)
	}
	
	// Test edge case priorities
	validTicket.Priority = 1
	if err := validTicket.Validate(); err != nil {
		t.Errorf("Priority 1 should be valid, got: %v", err)
	}
	
	validTicket.Priority = 5
	if err := validTicket.Validate(); err != nil {
		t.Errorf("Priority 5 should be valid, got: %v", err)
	}
}

func TestToYAML(t *testing.T) {
	ticket := &Ticket{
		ID:          "test-123",
		Title:       "Test ticket",
		Description: "A test ticket",
		Priority:    2,
		Locks:       []string{"test-lock"},
		Tags:        []string{"test"},
		CreatedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	
	yamlBytes, err := ticket.ToYAML()
	if err != nil {
		t.Fatalf("Failed to convert to YAML: %v", err)
	}
	
	// Parse it back to verify round-trip
	parsedTicket, err := LoadFromBytes(yamlBytes)
	if err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}
	
	if parsedTicket.ID != ticket.ID {
		t.Errorf("Round-trip failed for ID: expected %s, got %s", ticket.ID, parsedTicket.ID)
	}
	
	if parsedTicket.Title != ticket.Title {
		t.Errorf("Round-trip failed for Title: expected %s, got %s", ticket.Title, parsedTicket.Title)
	}
}