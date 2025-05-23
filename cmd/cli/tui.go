package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/config"
	"github.com/brettsmith212/amp-orchestrator/internal/ipc"
)

// startTUI starts the text-based user interface
func startTUI() {
	// Load configuration to get IPC socket path
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you're in a directory with config.yaml\n")
		os.Exit(1)
	}

	// Create IPC client
	ipcSocketPath := cfg.IPC.SocketPath
	if ipcSocketPath == "" {
		ipcSocketPath = "~/.orchestrator.sock"
	}

	fmt.Println("🔌 Connecting to orchestrator daemon...")
	client := ipc.NewClient(ipcSocketPath)
	
	if err := client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect to daemon: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure the orchestrator daemon is running\n")
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("✅ Connected to daemon!")
	fmt.Println("📊 Real-time orchestrator status (press Ctrl+C to exit):")
	fmt.Println()

	// Simple text-based display
	eventCount := 0
	for event := range client.Events() {
		eventCount++
		timestamp := event.Timestamp.Format("15:04:05")
		
		switch event.Type {
		case ipc.EventTypeQueueUpdated:
			if queueEvent, ok := event.Data.(map[string]interface{}); ok {
				queueLength := int(queueEvent["queue_length"].(float64))
				fmt.Printf("[%s] 📋 Queue: %d tickets pending\n", timestamp, queueLength)
			}
			
		case ipc.EventTypeTicketEnqueued:
			if ticketEvent, ok := event.Data.(map[string]interface{}); ok {
				if ticket, ok := ticketEvent["ticket"].(map[string]interface{}); ok {
					ticketID := ticket["id"].(string)
					title := ticket["title"].(string)
					fmt.Printf("[%s] 🎫 Enqueued: %s - %s\n", timestamp, ticketID, title)
				}
			}
			
		case ipc.EventTypeTicketStarted:
			if ticketEvent, ok := event.Data.(map[string]interface{}); ok {
				if ticket, ok := ticketEvent["ticket"].(map[string]interface{}); ok {
					ticketID := ticket["id"].(string)
					title := ticket["title"].(string)
					workerID := int(ticketEvent["worker_id"].(float64))
					fmt.Printf("[%s] 🚀 Worker %d started: %s - %s\n", timestamp, workerID, ticketID, title)
				}
			}
			
		case ipc.EventTypeTicketComplete:
			if ticketEvent, ok := event.Data.(map[string]interface{}); ok {
				if ticket, ok := ticketEvent["ticket"].(map[string]interface{}); ok {
					ticketID := ticket["id"].(string)
					title := ticket["title"].(string)
					workerID := int(ticketEvent["worker_id"].(float64))
					fmt.Printf("[%s] ✅ Worker %d completed: %s - %s\n", timestamp, workerID, ticketID, title)
				}
			}
			
		case ipc.EventTypeWorkerStatus:
			if workerEvent, ok := event.Data.(map[string]interface{}); ok {
				workerID := int(workerEvent["worker_id"].(float64))
				status := workerEvent["status"].(string)
				message := workerEvent["message"].(string)
				
				var icon string
				switch status {
				case "idle":
					icon = "😴"
				case "working":
					icon = "⚙️"
				case "error":
					icon = "❌"
				default:
					icon = "🤖"
				}
				
				fmt.Printf("[%s] %s Worker %d: %s - %s\n", timestamp, icon, workerID, status, message)
			}
		}

		// Add a separator every 10 events for readability
		if eventCount%10 == 0 {
			fmt.Println(strings.Repeat("-", 60))
		}
	}
}

// For testing: a simple function to simulate events
func testTUI() {
	client := ipc.NewClient("~/.orchestrator.sock")
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Listening for events...")
	timeout := time.After(10 * time.Second)
	
	for {
		select {
		case event := <-client.Events():
			fmt.Printf("Received event: %s at %s\n", event.Type, event.Timestamp.Format(time.RFC3339))
		case <-timeout:
			fmt.Println("Test timeout reached")
			return
		}
	}
}