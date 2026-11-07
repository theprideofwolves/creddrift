package store

import (
	"github.com/theprideofwolves/creddrift/config"
	"github.com/theprideofwolves/creddrift/scanner"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRescanMoveDeleteReappearAndScope(t *testing.T) {
	root := t.TempDir()
	db, e := NewStore(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	cfg := &config.Config{ScanTargets: []string{root}, IgnoreDirs: []string{"ignored"}}
	s := scanner.NewScanner(cfg)
	p := filepath.Join(root, "sample.txt")
	raw := "synthetic-test-only"
	os.WriteFile(p, []byte("password="+raw+"\n"), 0600)
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
	scan()
	rows, e := db.GetAllCredentials()
	if e != nil || len(rows) != 1 || rows[0].LastRotated != "Not recorded" {
		t.Fatal(rows, e)
	}
	os.WriteFile(p, []byte("\npassword="+raw+"\n"), 0600)
	scan()
	rows, _ = db.GetAllCredentials()
	if len(rows) != 1 || rows[0].LineNumber != 2 {
		t.Fatal(rows)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	db.SaveMatch(raw, "Generic", 3, outside, 1)
	os.Remove(p)
	scan()
	rows, _ = db.GetAllCredentials()
	if len(rows) != 1 || rows[0].FilePath != outside {
		t.Fatal("outside scope modified", rows)
	}
	os.WriteFile(p, []byte("password="+raw), 0600)
	scan()
	rows, _ = db.GetAllCredentials()
	if len(rows) != 2 {
		t.Fatal(rows)
	}
	db.IgnoreSecret(HashSecret(raw))
	scan()
	rows, _ = db.GetAllCredentials()
	if len(rows) != 0 {
		t.Fatal("ignore lost")
	}
	ignored, _ := db.GetIgnoredCredentials()
	if len(ignored) != 2 {
		t.Fatal(ignored)
	}
}
func TestIgnoredAndBinaryDoNotProveAbsence(t *testing.T) {
	root := t.TempDir()
	db, _ := NewStore(filepath.Join(t.TempDir(), "db"))
	defer db.Close()
	p := filepath.Join(root, "sample.bin")
	os.WriteFile(p, []byte{0}, 0600)
	db.SaveMatch("synthetic", "Generic", 3, p, 1)
	ignored := filepath.Join(root, "ignored", "old.txt")
	db.SaveMatch("synthetic", "Generic", 3, ignored, 1)
	if e := db.Reconcile(root, nil, nil, []string{"ignored"}); e != nil {
		t.Fatal(e)
	}
	rows, _ := db.GetAllCredentials()
	if len(rows) != 2 {
		t.Fatal(rows)
	}
}
func TestScanErrorDoesNotReconcile(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "sample.txt")
	os.WriteFile(p, []byte("password=synthetic-only"), 0600)
	db, _ := NewStore(filepath.Join(t.TempDir(), "db"))
	defer db.Close()
	s := scanner.NewScanner(&config.Config{ScanTargets: []string{root}})
	m, e := s.Scan()
	if e != nil {
		t.Fatal(e)
	}
	db.Reconcile(root, s.ScannedFiles, m, nil)
	os.WriteFile(p, []byte(strings.Repeat("x", 2<<20)), 0600)
	if _, e = s.Scan(); e == nil {
		t.Fatal("expected incomplete scan")
	}
	rows, _ := db.GetAllCredentials()
	if len(rows) != 1 {
		t.Fatal("old evidence lost")
	}
}
func TestReconcileRollsBackInvalidCoverage(t *testing.T) {
	root := t.TempDir()
	db, _ := NewStore(filepath.Join(t.TempDir(), "db"))
	defer db.Close()
	p := filepath.Join(root, "old.txt")
	db.SaveMatch("synthetic", "Generic", 3, p, 1)
	if e := db.Reconcile(root, nil, []scanner.Match{{File: filepath.Join(root, "uncovered.txt"), MatchTxt: "never-persist-raw"}}, nil); e == nil {
		t.Fatal("invalid coverage accepted")
	}
	rows, _ := db.GetAllCredentials()
	if len(rows) != 1 {
		t.Fatal("transaction not rolled back")
	}
}
func TestLegacyDiscoveryTimestampIsNotRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	s, e := NewStore(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SaveMatch("synthetic-legacy-only", "Generic", 3, "legacy.txt", 1); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("ALTER TABLE credentials DROP COLUMN rotation_recorded_at"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = NewStore(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	rows, e := s.GetAllCredentials()
	if e != nil || len(rows) != 1 || rows[0].LastRotated != "Not recorded" {
		t.Fatal(rows, e)
	}
	history, e := s.GetRotationHistory()
	if e != nil || len(history) != 0 {
		t.Fatal(history, e)
	}
	if e = s.MarkRotated(HashSecret("synthetic-legacy-only")); e != nil {
		t.Fatal(e)
	}
	history, e = s.GetRotationHistory()
	if e != nil || len(history) != 1 {
		t.Fatal(history, e)
	}
}

func TestLegacyRelativeLocationNeverProvesAbsence(t *testing.T) {
	s, e := NewStore(filepath.Join(t.TempDir(), "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.SaveMatch("synthetic-relative-only", "Generic", 3, "unknown-original-root.txt", 1); e != nil {
		t.Fatal(e)
	}
	root, e := os.Getwd()
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Reconcile(root, nil, nil, nil); e != nil {
		t.Fatal(e)
	}
	rows, e := s.GetAllCredentials()
	if e != nil || len(rows) != 1 {
		t.Fatal("legacy relative evidence was incorrectly retired", rows, e)
	}
}
