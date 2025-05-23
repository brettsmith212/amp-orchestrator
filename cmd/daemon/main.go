package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/brettsmith212/amp-orchestrator/internal/config"
	"github.com/brettsmith212/amp-orchestrator/internal/ipc"
	"github.com/brettsmith212/amp-orchestrator/internal/queue"
	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
	"github.com/brettsmith212/amp-orchestrator/internal/watch"
	"github.com/brettsmith212/amp-orchestrator/internal/worker"
	"github.com/brettsmith212/amp-orchestrator/pkg/gitutils"
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

	// Create working directory if it doesn't exist
	if err := os.MkdirAll(cfg.Repository.Workdir, 0755); err != nil {
		log.Fatalf("Failed to create working directory: %v", err)
	}

	// Initialize git repository if needed
	repo := gitutils.NewRepo(cfg.Repository.Path)
	if _, err := os.Stat(cfg.Repository.Path); os.IsNotExist(err) {
		log.Printf("Creating bare repository at %s", cfg.Repository.Path)
		if err := gitutils.InitBareRepo(cfg.Repository.Path); err != nil {
			log.Fatalf("Failed to create bare repository: %v", err)
		}
	}

	// Check if repository has any commits, create initial commit if needed
	branches, err := repo.ListBranches()
	if err != nil || len(branches) == 0 {
		log.Printf("Creating initial commit in repository")
		if err := repo.CreateInitialCommit(); err != nil {
			log.Fatalf("Failed to create initial commit: %v", err)
		}
	}

	// Install git hooks for CI integration
	if err := installGitHooks(cfg.Repository.Path); err != nil {
		log.Printf("Warning: Failed to install git hooks: %v", err)
	} else {
		log.Printf("Installed git hooks for CI integration")
	}

	// Initialize priority queue
	ticketQueue := queue.New()
	log.Printf("Initialized ticket queue")

	// Initialize IPC server
	ipcSocketPath := cfg.IPC.SocketPath
	if ipcSocketPath == "" {
		ipcSocketPath = "~/.orchestrator.sock"
	}
	ipcServer := ipc.NewServer(ipcSocketPath)
	if err := ipcServer.Start(); err != nil {
		log.Printf("Warning: Failed to start IPC server: %v", err)
		ipcServer = nil
	} else {
		log.Printf("Started IPC server on %s", ipcSocketPath)
	}

	// Initialize backlog watcher
	watcherConfig := watch.Config{
		BacklogPath:    cfg.Scheduler.BacklogPath,
		TickerInterval: time.Duration(cfg.Scheduler.PollInterval) * time.Second,
	}

	watcher, err := watch.New(watcherConfig, ticketQueue)
	if err != nil {
		log.Fatalf("Failed to create backlog watcher: %v", err)
	}

	// Set up IPC event publishing for watcher
	if ipcServer != nil {
		watcher.SetEventPublisher(func(t *ticket.Ticket) {
			ipcServer.PublishTicketEnqueued(t)
			// Also publish queue update
			var nextTicket *ticket.Ticket
			if ticketQueue.Len() > 0 {
				nextTicket = ticketQueue.Peek()
			}
			ipcServer.PublishQueueUpdated(ticketQueue.Len(), nextTicket)
		})
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

	// Start workers
	workers := make([]*worker.Worker, cfg.Agents.Count)
	for i := 0; i < cfg.Agents.Count; i++ {
		workerConfig := worker.Config{
			ID:          i + 1,
			RepoPath:    cfg.Repository.Path,
			WorkDir:     cfg.Repository.Workdir,
			CIStatusDir: cfg.CI.StatusPath,
		}
		
		workers[i] = worker.New(workerConfig, ticketQueue)
		
		// Set up IPC event publishing for worker
		if ipcServer != nil {
			workers[i].SetEventPublisher(func(eventType string, workerID int, t *ticket.Ticket, message string) {
				switch eventType {
				case "started":
					ipcServer.PublishTicketStarted(t, workerID)
					ipcServer.PublishWorkerStatus(workerID, "working", t, message)
				case "completed":
					ipcServer.PublishTicketComplete(t, workerID)
					ipcServer.PublishWorkerStatus(workerID, "idle", nil, message)
				}
			})
		}
		
		// Start each worker in its own goroutine
		go func(w *worker.Worker) {
			log.Printf("Starting worker %d...", w.GetStatus().ID)
			if err := w.Start(ctx); err != nil {
				log.Printf("Worker %d stopped: %v", w.GetStatus().ID, err)
			}
		}(workers[i])
	}

	// Log periodic queue and worker status
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
				
				// Log worker status
				for _, w := range workers {
					status := w.GetStatus()
					if status.CurrentTicket != nil {
						log.Printf("Worker %d: processing %s (%s)", 
							status.ID, status.CurrentTicket.ID, status.CurrentTicket.Title)
					} else {
						log.Printf("Worker %d: idle", status.ID)
					}
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

	// Stop IPC server
	if ipcServer != nil {
		if err := ipcServer.Stop(); err != nil {
			log.Printf("Error stopping IPC server: %v", err)
		}
	}

	// Give components time to shut down gracefully
	time.Sleep(1 * time.Second)
	log.Printf("Orchestrator stopped")
}

// installGitHooks installs the post-receive hook for CI integration
func installGitHooks(repoPath string) error {
	// Find the ci.sh script path (relative to the daemon executable)
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}
	
	// Assume ci.sh is in the project root (parent of bin/)
	projectRoot := filepath.Dir(filepath.Dir(execPath))
	ciScriptPath := filepath.Join(projectRoot, "ci.sh")
	
	// Check if ci.sh exists, if not use the current directory
	if _, err := os.Stat(ciScriptPath); os.IsNotExist(err) {
		// Fall back to current working directory
		ciScriptPath = "ci.sh"
	}
	
	// Run the hook installer
	cmd := exec.Command("go", "run", 
		filepath.Join(projectRoot, "scripts", "install_hook.go"),
		"--repo", repoPath,
		"--ci-script", ciScriptPath)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook installation failed: %w: %s", err, output)
	}
	
	return nil
}