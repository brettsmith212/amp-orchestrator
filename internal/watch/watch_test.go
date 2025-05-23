package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/queue"
)

func TestWatcherFileEvent(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	
	// Create a queue
	q := queue.New()
	
	// Create watcher config
	config := Config{
		BacklogPath:    tmpDir,
		TickerInterval: 100 * time.Millisecond,
	}
	
	// Create watcher
	watcher, err := New(config, q)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()
	
	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		if err := watcher.Start(ctx); err != nil {
			t.Logf("Watcher error: %v", err)
		}
	}()
	
	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)
	
	// Create a test ticket file
	ticketYAML := `id: "test-watch-123"
title: "Test watcher ticket"
description: "A ticket to test the watcher"
priority: 1`
	
	testFile := filepath.Join(tmpDir, "test-ticket.yaml")
	if err := os.WriteFile(testFile, []byte(ticketYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Wait for watcher to process the file
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for ticket to be enqueued")
		case <-ticker.C:
			if q.Len() > 0 {
				ticket := q.Peek()
				if ticket != nil && ticket.ID == "test-watch-123" {
					t.Logf("Successfully detected and enqueued ticket: %s", ticket.ID)
					return
				}
			}
		}
	}
}

func TestWatcherTickerFallback(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	
	// Create a queue
	q := queue.New()
	
	// Create watcher config with very frequent ticker
	config := Config{
		BacklogPath:    tmpDir,
		TickerInterval: 50 * time.Millisecond,
	}
	
	// Create a test ticket file BEFORE starting the watcher
	ticketYAML := `id: "test-ticker-456"
title: "Test ticker ticket"
description: "A ticket to test the ticker fallback"
priority: 2`
	
	testFile := filepath.Join(tmpDir, "existing-ticket.yaml")
	if err := os.WriteFile(testFile, []byte(ticketYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Create watcher
	watcher, err := New(config, q)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()
	
	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		if err := watcher.Start(ctx); err != nil {
			t.Logf("Watcher error: %v", err)
		}
	}()
	
	// Wait for ticker to fire at least once
	timeout := time.After(500 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for ticker to process existing file")
		case <-ticker.C:
			if q.Len() > 0 {
				ticket := q.Peek()
				if ticket != nil && ticket.ID == "test-ticker-456" {
					t.Logf("Ticker successfully found and enqueued existing ticket: %s", ticket.ID)
					return
				}
			}
		}
	}
}

func TestWatcherIgnoresDuplicates(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	
	// Create a queue
	q := queue.New()
	
	// Create watcher config
	config := Config{
		BacklogPath:    tmpDir,
		TickerInterval: 50 * time.Millisecond,
	}
	
	// Create watcher
	watcher, err := New(config, q)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()
	
	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		if err := watcher.Start(ctx); err != nil {
			t.Logf("Watcher error: %v", err)
		}
	}()
	
	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)
	
	// Create a test ticket file
	ticketYAML := `id: "test-duplicate-789"
title: "Test duplicate ticket"
description: "A ticket to test duplicate detection"
priority: 3`
	
	testFile := filepath.Join(tmpDir, "duplicate-ticket.yaml")
	if err := os.WriteFile(testFile, []byte(ticketYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Wait for first processing
	time.Sleep(200 * time.Millisecond)
	
	// Modify the file (should trigger another event but not duplicate enqueue)
	updatedYAML := `id: "test-duplicate-789"
title: "Test duplicate ticket (updated)"
description: "A ticket to test duplicate detection (updated)"
priority: 3`
	
	if err := os.WriteFile(testFile, []byte(updatedYAML), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}
	
	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	
	// Should still only have one ticket in queue
	if q.Len() != 1 {
		t.Errorf("Expected queue length 1 (no duplicates), got %d", q.Len())
	}
	
	ticket := q.Peek()
	if ticket == nil || ticket.ID != "test-duplicate-789" {
		t.Error("Expected the duplicate ticket to be detected and not re-enqueued")
	}
}

func TestWatcherInvalidYAML(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	
	// Create a queue
	q := queue.New()
	
	// Create watcher config
	config := Config{
		BacklogPath:    tmpDir,
		TickerInterval: 50 * time.Millisecond,
	}
	
	// Create watcher
	watcher, err := New(config, q)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()
	
	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		if err := watcher.Start(ctx); err != nil {
			t.Logf("Watcher error: %v", err)
		}
	}()
	
	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)
	
	// Create invalid YAML file
	invalidYAML := `id: "invalid"
title: "Invalid ticket
description: "Missing quote and invalid YAML"
priority: abc`
	
	testFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(testFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	
	// Queue should remain empty due to invalid YAML
	if q.Len() != 0 {
		t.Errorf("Expected queue to be empty for invalid YAML, got %d items", q.Len())
	}
}

func TestWatcherNonYAMLFiles(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	
	// Create a queue
	q := queue.New()
	
	// Create watcher config
	config := Config{
		BacklogPath:    tmpDir,
		TickerInterval: 50 * time.Millisecond,
	}
	
	// Create watcher
	watcher, err := New(config, q)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()
	
	// Start watcher in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		if err := watcher.Start(ctx); err != nil {
			t.Logf("Watcher error: %v", err)
		}
	}()
	
	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)
	
	// Create non-YAML files
	textFile := filepath.Join(tmpDir, "not-a-ticket.txt")
	if err := os.WriteFile(textFile, []byte("This is not a YAML file"), 0644); err != nil {
		t.Fatalf("Failed to write text file: %v", err)
	}
	
	jsonFile := filepath.Join(tmpDir, "not-a-ticket.json")
	if err := os.WriteFile(jsonFile, []byte(`{"id": "test"}`), 0644); err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}
	
	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	
	// Queue should remain empty for non-YAML files
	if q.Len() != 0 {
		t.Errorf("Expected queue to be empty for non-YAML files, got %d items", q.Len())
	}
}