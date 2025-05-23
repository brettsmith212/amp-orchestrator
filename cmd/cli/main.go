package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/brettsmith212/amp-orchestrator/internal/ticket"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "validate":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s validate <ticket-file.yaml>\n", os.Args[0])
			os.Exit(1)
		}
		validateTicket(os.Args[2])
		
	case "enqueue":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s enqueue <ticket-file.yaml>\n", os.Args[0])
			os.Exit(1)
		}
		enqueueTicket(os.Args[2])
		
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [args]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  validate <file>  Validate a ticket YAML file\n")
	fmt.Fprintf(os.Stderr, "  enqueue <file>   Enqueue a ticket by copying it to the backlog directory\n")
}

func validateTicket(filePath string) {
	// Load and validate the ticket
	t, err := ticket.Load(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Ticket validation passed\n")
	fmt.Printf("   ID: %s\n", t.ID)
	fmt.Printf("   Title: %s\n", t.Title)
	fmt.Printf("   Priority: %d\n", t.Priority)
	if len(t.Locks) > 0 {
		fmt.Printf("   Locks: %v\n", t.Locks)
	}
	if len(t.Dependencies) > 0 {
		fmt.Printf("   Dependencies: %v\n", t.Dependencies)
	}
}

func enqueueTicket(filePath string) {
	// First validate the ticket
	t, err := ticket.Load(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load ticket: %v\n", err)
		os.Exit(1)
	}
	
	// Determine backlog directory
	// Default to ./backlog, but could be made configurable
	backlogDir := "./backlog"
	if envDir := os.Getenv("ORCHESTRATOR_BACKLOG_PATH"); envDir != "" {
		backlogDir = envDir
	}
	
	// Create backlog directory if it doesn't exist
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create backlog directory: %v\n", err)
		os.Exit(1)
	}
	
	// Determine destination filename
	originalName := filepath.Base(filePath)
	destPath := filepath.Join(backlogDir, originalName)
	
	// Check if destination already exists and has the same ticket ID
	if _, err := os.Stat(destPath); err == nil {
		// File exists, check if it's the same ticket
		existingTicket, loadErr := ticket.Load(destPath)
		if loadErr == nil && existingTicket.ID == t.ID {
			fmt.Printf("⚠️  Ticket %s is already in the backlog\n", t.ID)
			return
		}
		
		// Different ticket with same filename, need to rename
		ext := filepath.Ext(originalName)
		base := originalName[:len(originalName)-len(ext)]
		for i := 1; ; i++ {
			newName := fmt.Sprintf("%s-%d%s", base, i, ext)
			destPath = filepath.Join(backlogDir, newName)
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
		}
	}
	
	// Read source file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read source file: %v\n", err)
		os.Exit(1)
	}
	
	// Write to backlog directory
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to write to backlog: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Enqueued ticket %s\n", t.ID)
	fmt.Printf("   File: %s\n", destPath)
	fmt.Printf("   Title: %s\n", t.Title)
	fmt.Printf("   Priority: %d\n", t.Priority)
	
	log.Printf("Enqueued ticket %s: %s", t.ID, t.Title)
}