package scanner

import (
	"context"
	"github.com/theprideofwolves/creddrift/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatternsLocationsAndIgnoredFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "ignored"), 0700); err != nil {
		t.Fatal(err)
	}
	data := "aws = " + "AKIA" + strings.Repeat("A", 16) + "\ngithub = " + "ghp_" + strings.Repeat("b", 36) + "\npassword = demo-password\n"
	for path, contents := range map[string][]byte{"sample.txt": []byte(data), "ignored/skip.txt": []byte(data), "binary.bin": append([]byte{0}, []byte(data)...)} {
		if err := os.WriteFile(filepath.Join(dir, path), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := NewScanner(&config.Config{ScanTargets: []string{dir}, IgnoreDirs: []string{"ignored"}})
	matches, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 3 {
		t.Fatalf("matches=%+v", matches)
	}
	for i, m := range matches {
		if m.LineNum != i+1 || m.File != filepath.Join(dir, "sample.txt") {
			t.Fatalf("wrong location %+v", m)
		}
	}
}
func TestScanReportsLongLineError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "long.txt")
	if err := os.WriteFile(p, []byte(strings.Repeat("a", 2*1024*1024)), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := NewScanner(&config.Config{ScanTargets: []string{p}}).Scan()
	if err == nil {
		t.Fatal("silently accepted truncated scan")
	}
}
func TestSymlinkIsOutsideCoverage(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if e := os.WriteFile(outside, []byte("password=synthetic-only"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(root, "link.txt")); e != nil {
		t.Skip("symlink privilege unavailable")
	}
	s := NewScanner(&config.Config{ScanTargets: []string{root}})
	m, e := s.Scan()
	if e != nil || len(m) != 0 || len(s.ScannedFiles) != 0 {
		t.Fatal(m, e)
	}
}

func TestCancelledScan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewScanner(&config.Config{ScanTargets: []string{t.TempDir()}})
	if _, e := s.ScanContext(ctx); e == nil {
		t.Fatal("cancelled scan appeared complete")
	}
}
