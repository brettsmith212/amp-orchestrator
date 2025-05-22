package main

import (
	"fmt"
	"log"
	"os"

	"github.com/brettsmith212/amp-orchestrator/internal/config"
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

	// TODO: Initialize other components in subsequent steps
	log.Printf("Orchestrator initialized and ready")
}