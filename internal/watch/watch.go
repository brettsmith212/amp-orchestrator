package watch

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/brettsmith212/amp-orchestrator/internal/queue"
	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
)

// Watcher monitors a directory for new ticket files and enqueues them
type Watcher struct {
	backlogPath string
	queue       *queue.Queue
	tickerInterval time.Duration
	fsWatcher   *fsnotify.Watcher
}

// Config holds watcher configuration
type Config struct {
	BacklogPath    string
	TickerInterval time.Duration
}

// New creates a new backlog watcher
func New(config Config, q *queue.Queue) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		backlogPath:    config.BacklogPath,
		queue:          q,
		tickerInterval: config.TickerInterval,
		fsWatcher:      fsWatcher,
	}, nil
}

// Start begins watching the backlog directory for changes
func (w *Watcher) Start(ctx context.Context) error {
	// Add the backlog directory to the watcher
	err := w.fsWatcher.Add(w.backlogPath)
	if err != nil {
		return fmt.Errorf("failed to add directory to watcher: %w", err)
	}

	log.Printf("Started backlog watcher on %s", w.backlogPath)

	// Start the ticker for periodic scans
	ticker := time.NewTicker(w.tickerInterval)
	defer ticker.Stop()

	// Initial scan of existing files
	if err := w.scanDirectory(); err != nil {
		log.Printf("Error during initial scan: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping backlog watcher")
			return w.fsWatcher.Close()

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}
			w.handleFileEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}
			log.Printf("Watcher error: %v", err)

		case <-ticker.C:
			// Periodic scan as fallback
			if err := w.scanDirectory(); err != nil {
				log.Printf("Error during periodic scan: %v", err)
			}
		}
	}
}

// handleFileEvent processes file system events
func (w *Watcher) handleFileEvent(event fsnotify.Event) {
	// Only process write and create events for YAML files
	if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
		if w.isTicketFile(event.Name) {
			log.Printf("File event: %s %s", event.Op, event.Name)
			w.processTicketFile(event.Name)
		}
	}
}

// scanDirectory scans the backlog directory for ticket files
func (w *Watcher) scanDirectory() error {
	pattern := filepath.Join(w.backlogPath, "*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	for _, file := range matches {
		w.processTicketFile(file)
	}

	// Also check for .yml files
	pattern = filepath.Join(w.backlogPath, "*.yml")
	matches, err = filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to scan directory for .yml files: %w", err)
	}

	for _, file := range matches {
		w.processTicketFile(file)
	}

	return nil
}

// processTicketFile attempts to load and enqueue a ticket file
func (w *Watcher) processTicketFile(filepath string) {
	log.Printf("Processing ticket file: %s", filepath)

	ticket, err := ticket.Load(filepath)
	if err != nil {
		log.Printf("Failed to load ticket from %s: %v", filepath, err)
		return
	}

	// Check if ticket is already in queue to avoid duplicates
	if w.isTicketInQueue(ticket.ID) {
		log.Printf("Ticket %s is already in queue, skipping", ticket.ID)
		return
	}

	w.queue.Push(ticket)
	log.Printf("Enqueued ticket %s: %s", ticket.ID, ticket.Title)

	// Move the file to a processed directory to avoid re-processing
	if err := w.moveToProcessed(filepath); err != nil {
		log.Printf("Failed to move processed file %s: %v", filepath, err)
	}
}

// isTicketInQueue checks if a ticket with the given ID is already in the queue
func (w *Watcher) isTicketInQueue(ticketID string) bool {
	tickets := w.queue.List()
	for _, t := range tickets {
		if t.ID == ticketID {
			return true
		}
	}
	return false
}

// isTicketFile checks if the file is a YAML ticket file
func (w *Watcher) isTicketFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".yaml" || ext == ".yml"
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	return w.fsWatcher.Close()
}

// GetQueueStatus returns information about the current queue state
func (w *Watcher) GetQueueStatus() string {
	return w.queue.String()
}

// moveToProcessed moves a processed ticket file to a processed subdirectory
func (w *Watcher) moveToProcessed(filePath string) error {
	// Create processed directory if it doesn't exist
	processedDir := filepath.Join(w.backlogPath, "processed")
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	// Get the filename
	filename := filepath.Base(filePath)
	
	// Move file to processed directory
	destPath := filepath.Join(processedDir, filename)
	if err := os.Rename(filePath, destPath); err != nil {
		return fmt.Errorf("failed to move file to processed directory: %w", err)
	}

	log.Printf("Moved processed ticket file to %s", destPath)
	return nil
}