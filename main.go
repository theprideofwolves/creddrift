package main

import (
	"fmt"
	"github.com/theprideofwolves/creddrift/config"
	"github.com/theprideofwolves/creddrift/internal/desktop"
	"github.com/theprideofwolves/creddrift/internal/store"
	"github.com/theprideofwolves/creddrift/internal/ui"
	"github.com/theprideofwolves/creddrift/scanner"
	"os"
	"path/filepath"
)

func run() error {
	dir, e := desktop.DataDir("CredDrift")
	if e != nil {
		return e
	}
	db, e := store.NewStore(filepath.Join(dir, "creddrift.db"))
	if e != nil {
		return e
	}
	defer db.Close()
	cfg := &config.Config{IgnoreDirs: []string{".git", "node_modules", "vendor", "CredDrift"}}
	s := scanner.NewScanner(cfg)
	return ui.Serve(db, cfg, s, dir)
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		fmt.Println("Press Enter to close")
		fmt.Scanln()
		os.Exit(1)
	}
}
