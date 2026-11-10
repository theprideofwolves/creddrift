package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprideofwolves/creddrift/config"
)

func TestEmptyMissingAndOversizedScopes(t *testing.T) {
	root := t.TempDir()
	s := NewScanner(&config.Config{ScanTargets: []string{root}})
	if m, err := s.Scan(); err != nil || len(m) != 0 {
		t.Fatal("empty folder", err)
	}
	missing := NewScanner(&config.Config{ScanTargets: []string{filepath.Join(root, "missing")}})
	if _, err := missing.Scan(); err == nil {
		t.Fatal("missing scope succeeded")
	}
	f, err := os.Create(filepath.Join(root, "large.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate((32 << 20) + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()
	if _, err := s.Scan(); err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Fatal("oversize file accepted", err)
	}
}

func TestFileCountLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 50001; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("%05d.txt", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := NewScanner(&config.Config{ScanTargets: []string{root}})
	if _, err := s.Scan(); err == nil || !strings.Contains(err.Error(), "50000") {
		t.Fatal("file count limit not enforced", err)
	}
}

func TestPatternBoundariesAndAssignmentVariants(t *testing.T) {
	root := t.TempDir()
	aws, gh := "AKIA"+strings.Repeat("A", 16), "ghp_"+strings.Repeat("b", 36)
	data := strings.Join([]string{
		aws + "X", "X" + aws, gh + "x", "x" + gh,
		"API_KEY: 'synthetic-quoted-fixture'", "PASSWORD = \"synthetic-double-fixture\"",
		"password=", "password=\"\"", aws + " " + gh,
	}, "\r\n")
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := NewScanner(&config.Config{ScanTargets: []string{root}}).Scan()
	if err != nil || len(m) != 4 {
		t.Fatalf("pattern boundary count=%d err=%v", len(m), err)
	}
	for i, line := range []int{5, 6, 9, 9} {
		if m[i].LineNum != line {
			t.Fatal("wrong CRLF location")
		}
	}
}

func TestCandidateLimitStopsWithinFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "many.txt"), []byte(strings.Repeat("password=synthetic-limit-fixture\n", 100010)), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := NewScanner(&config.Config{ScanTargets: []string{root}}).Scan()
	if err == nil || !strings.Contains(err.Error(), "100000") {
		t.Fatal("candidate limit not enforced", err)
	}
	if len(m) > 100000 {
		t.Fatalf("retained %d candidates beyond the limit", len(m))
	}
}
