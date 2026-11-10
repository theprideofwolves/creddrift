package store

import (
	"github.com/theprideofwolves/creddrift/config"
	"github.com/theprideofwolves/creddrift/scanner"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceLifecycleAndRestart(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "db")
	db, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{ScanTargets: []string{root}, IgnoreDirs: []string{"excluded"}}
	s := scanner.NewScanner(cfg)
	raw := "synthetic-evidence-fixture"
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	write := func(p string) {
		t.Helper()
		if err := os.WriteFile(p, []byte("password="+raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	scan := func() {
		t.Helper()
		m, e := s.Scan()
		if e != nil {
			t.Fatal(e)
		}
		if e = db.Reconcile(root, s.ScannedFiles, m, cfg.IgnoreDirs); e != nil {
			t.Fatal(e)
		}
	}
	states := func(want map[string]int) {
		t.Helper()
		history, e := db.Timeline(HashSecret(raw))
		if e != nil {
			t.Fatal(e)
		}
		got := map[string]int{}
		for _, event := range history[0].Events {
			got[event.State]++
		}
		for state, n := range want {
			if got[state] != n {
				t.Fatalf("state %s got %d want %d", state, got[state], n)
			}
		}
		if len(got) != len(want) {
			t.Fatal(got)
		}
	}
	write(first)
	scan()
	states(map[string]int{"new": 1})
	write(second)
	scan()
	states(map[string]int{"new": 1, "still present": 1})
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	scan()
	states(map[string]int{"absent from checked coverage": 1, "still present": 1})
	write(first)
	scan()
	states(map[string]int{"reappeared": 1, "still present": 1})
	if err := os.WriteFile(second, []byte{0}, 0600); err != nil {
		t.Fatal(err)
	}
	scan()
	states(map[string]int{"unverified": 1, "still present": 1})
	h, _ := db.Timeline("")
	if !strings.Contains(h[0].Events[1].Reason, "binary") {
		t.Fatal("missing binary coverage reason")
	}
	if err := db.RecordIncomplete(root, "cancelled"); err != nil {
		t.Fatal(err)
	}
	h, _ = db.Timeline("")
	if h[0].Outcome != "cancelled" || len(h[0].Events) != 0 {
		t.Fatal("partial evidence committed")
	}
	before := len(h)
	if err := db.Reconcile(root, nil, []scanner.Match{{File: filepath.Join(root, "uncovered"), MatchTxt: raw}}, nil); err == nil {
		t.Fatal("expected rollback")
	}
	h, _ = db.Timeline("")
	if len(h) != before {
		t.Fatal("failed transaction committed history")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h, err = db.Timeline("")
	if err != nil || len(h) != before {
		t.Fatal("history not persisted", err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(bytes), raw) {
		t.Fatal("raw value persisted", err)
	}
}
func TestCoverageReasonsAndIdentityFilter(t *testing.T) {
	root := t.TempDir()
	db, err := NewStore(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	paths := []string{filepath.Join(root, "excluded", "old.txt"), filepath.Join(t.TempDir(), "outside.txt"), "legacy-relative.txt"}
	for i, p := range paths {
		if err := db.SaveMatch("synthetic-reason-fixture", "Generic", 3, p, i+1); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Reconcile(root, nil, nil, []string{"excluded"}); err != nil {
		t.Fatal(err)
	}
	h, err := db.Timeline("")
	if err != nil || len(h) != 1 || len(h[0].Events) != 3 {
		t.Fatal("missing retained events", err)
	}
	reasons := map[string]bool{}
	for _, e := range h[0].Events {
		if e.State != "unverified" {
			t.Fatal(e.State)
		}
		reasons[e.Reason] = true
	}
	for _, reason := range []string{"excluded directory", "outside selected scope", "legacy relative location"} {
		if !reasons[reason] {
			t.Fatal(reason)
		}
	}
	filtered, err := db.Timeline("not-an-existing-identity")
	if err != nil || len(filtered) != 1 || len(filtered[0].Events) != 0 {
		t.Fatal("identity filter failed", err)
	}
}
