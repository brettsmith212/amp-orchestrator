package ipc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
)

func TestIPCServer(t *testing.T) {
	// Create temporary directory for socket
	tmpDir, err := os.MkdirTemp("", "ipc-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create and start server
	server := NewServer(socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create and connect client
	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Give client time to connect
	time.Sleep(100 * time.Millisecond)

	// Test publishing an event
	testTicket := &ticket.Ticket{
		ID:       "test-001",
		Title:    "Test Ticket",
		Priority: 1,
	}

	server.PublishTicketEnqueued(testTicket)

	// Wait for event to be received
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case event := <-client.Events():
		if event.Type != EventTypeTicketEnqueued {
			t.Errorf("Expected event type %s, got %s", EventTypeTicketEnqueued, event.Type)
		}

		// Verify event data
		ticketEvent, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected event data to be map, got %T", event.Data)
		}

		ticketData, ok := ticketEvent["ticket"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected ticket data in event")
		}

		if ticketData["id"] != testTicket.ID {
			t.Errorf("Expected ticket ID %s, got %s", testTicket.ID, ticketData["id"])
		}

	case <-ctx.Done():
		t.Fatal("Timeout waiting for event")
	}
}

func TestIPCQueueEvent(t *testing.T) {
	// Create temporary directory for socket
	tmpDir, err := os.MkdirTemp("", "ipc-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create and start server
	server := NewServer(socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create and connect client
	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Give client time to connect
	time.Sleep(100 * time.Millisecond)

	// Test queue updated event
	nextTicket := &ticket.Ticket{
		ID:       "next-001",
		Title:    "Next Ticket",
		Priority: 1,
	}

	server.PublishQueueUpdated(5, nextTicket)

	// Wait for event to be received
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case event := <-client.Events():
		if event.Type != EventTypeQueueUpdated {
			t.Errorf("Expected event type %s, got %s", EventTypeQueueUpdated, event.Type)
		}

		// Verify event data
		queueEvent, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected event data to be map, got %T", event.Data)
		}

		queueLength, ok := queueEvent["queue_length"].(float64)
		if !ok || int(queueLength) != 5 {
			t.Errorf("Expected queue length 5, got %v", queueEvent["queue_length"])
		}

	case <-ctx.Done():
		t.Fatal("Timeout waiting for event")
	}
}

func TestIPCWorkerStatusEvent(t *testing.T) {
	// Create temporary directory for socket
	tmpDir, err := os.MkdirTemp("", "ipc-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create and start server
	server := NewServer(socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create and connect client
	client := NewClient(socketPath)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Give client time to connect
	time.Sleep(100 * time.Millisecond)

	// Test worker status event
	currentTicket := &ticket.Ticket{
		ID:       "current-001",
		Title:    "Current Ticket",
		Priority: 1,
	}

	server.PublishWorkerStatus(1, "working", currentTicket, "Processing ticket")

	// Wait for event to be received
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case event := <-client.Events():
		if event.Type != EventTypeWorkerStatus {
			t.Errorf("Expected event type %s, got %s", EventTypeWorkerStatus, event.Type)
		}

		// Verify event data
		workerEvent, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected event data to be map, got %T", event.Data)
		}

		workerID, ok := workerEvent["worker_id"].(float64)
		if !ok || int(workerID) != 1 {
			t.Errorf("Expected worker ID 1, got %v", workerEvent["worker_id"])
		}

		status, ok := workerEvent["status"].(string)
		if !ok || status != "working" {
			t.Errorf("Expected status 'working', got %v", workerEvent["status"])
		}

	case <-ctx.Done():
		t.Fatal("Timeout waiting for event")
	}
}

func TestIPCMultipleClients(t *testing.T) {
	// Create temporary directory for socket
	tmpDir, err := os.MkdirTemp("", "ipc-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create and start server
	server := NewServer(socketPath)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create multiple clients
	client1 := NewClient(socketPath)
	if err := client1.Connect(); err != nil {
		t.Fatalf("Failed to connect client1: %v", err)
	}
	defer client1.Close()

	client2 := NewClient(socketPath)
	if err := client2.Connect(); err != nil {
		t.Fatalf("Failed to connect client2: %v", err)
	}
	defer client2.Close()

	// Give clients time to connect
	time.Sleep(100 * time.Millisecond)

	// Publish event
	server.PublishQueueUpdated(3, nil)

	// Both clients should receive the event
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check client1
	select {
	case event := <-client1.Events():
		if event.Type != EventTypeQueueUpdated {
			t.Errorf("Client1: Expected event type %s, got %s", EventTypeQueueUpdated, event.Type)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for event on client1")
	}

	// Check client2
	select {
	case event := <-client2.Events():
		if event.Type != EventTypeQueueUpdated {
			t.Errorf("Client2: Expected event type %s, got %s", EventTypeQueueUpdated, event.Type)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for event on client2")
	}
}