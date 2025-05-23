package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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

	// Create and start the Bubble Tea program
	model := NewModel(client)
	program := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

