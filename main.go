package main

import (
	"fmt"
	"log"

	"github.com/varkuru/creddrift/config"
	"github.com/varkuru/creddrift/internal/store"
	"github.com/varkuru/creddrift/internal/ui"
	"github.com/varkuru/creddrift/scanner"
)

func main() {
	fmt.Println("CredDrift Initializing...")

	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Scanning targets: %v\n", cfg.ScanTargets)

	// Initialize Database Store
	dbStore, err := store.NewStore("creddrift.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbStore.Close()
	fmt.Println("SQLite database creddrift.db initialized.")

	// Build the scanner
	s := scanner.NewScanner(cfg)

	// Execute scan
	matches, err := s.Scan()
	if err != nil {
		log.Fatalf("Scanner encountered an error: %v", err)
	}

	// Output results and store them
	fmt.Printf("\nScan Complete. Found %d potential secrets.\n", len(matches))
	for _, m := range matches {
		fmt.Printf("[%s] Entropy: %.2f | File: %s:%d\n", m.Type, m.Entropy, m.File, m.LineNum)
		
		// Save to the database
		err := dbStore.SaveMatch(m.MatchTxt, m.Type, m.Entropy, m.File, m.LineNum)
		if err != nil {
			log.Printf("Failed to save match to database: %v", err)
		}
	}

	// Start the Web Dashboard
	ui.StartServer(dbStore, cfg, s, ":8080")
}
