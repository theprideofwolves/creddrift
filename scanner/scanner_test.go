package scanner

import (
	"github.com/varkuru/creddrift/config"
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
