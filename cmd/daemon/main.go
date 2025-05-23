package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/config"
	"github.com/brettsmith212/amp-orchestrator/internal/queue"
	"github.com/brettsmith212/amp-orchestrator/internal/watch"
)

func main() {
	fmt.Println("Amp Orchestrator daemon starting...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		log.Printf("Make sure you've copied config.sample.yaml to ~/.config/orchestrator/config.yaml")
		os.Exit(1)
	}

	// Log config loaded successfully
	log.Printf("Configuration loaded successfully")
	log.Printf("Repository path: %s", cfg.Repository.Path)
	log.Printf("Running with %d agents", cfg.Agents.Count)
	log.Printf("Backlog path: %s", cfg.Scheduler.BacklogPath)

	// Create backlog directory if it doesn't exist
	if err := os.MkdirAll(cfg.Scheduler.BacklogPath, 0755); err != nil {
		log.Fatalf("Failed to create backlog directory: %v", err)
	}

	// Initialize priority queue
	ticketQueue := queue.New()
	log.Printf("Initialized ticket queue")

	// Initialize backlog watcher
	watcherConfig := watch.Config{
		BacklogPath:    cfg.Scheduler.BacklogPath,
		TickerInterval: time.Duration(cfg.Scheduler.PollInterval) * time.Second,
	}

	watcher, err := watch.New(watcherConfig, ticketQueue)
	if err != nil {
		log.Fatalf("Failed to create backlog watcher: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start watcher in a goroutine
	go func() {
		log.Printf("Starting backlog watcher...")
		if err := watcher.Start(ctx); err != nil {
			log.Printf("Watcher stopped: %v", err)
		}
	}()

	// Log periodic queue status
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Printf("Queue status: %d tickets pending", ticketQueue.Len())
				if ticketQueue.Len() > 0 {
					log.Printf("Next ticket: %s", ticketQueue.Peek().ID)
				}
			}
		}
	}()

	log.Printf("Orchestrator initialized and ready")

	// Wait for shutdown signal
	<-sigChan
	log.Printf("Received shutdown signal, stopping...")

	// Cancel context to stop all goroutines
	cancel()

	// Give components time to shut down gracefully
	time.Sleep(1 * time.Second)
	log.Printf("Orchestrator stopped")
}