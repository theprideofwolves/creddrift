package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprideofwolves/creddrift/scanner"
)

func TestExcludedDirectoryNameIsNotExcludedFile(t *testing.T) {
	root := t.TempDir()
	s, err := NewStore(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := filepath.Join(root, "vendor")
	if err := s.SaveMatch("synthetic-name-fixture", "Generic", 3, p, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Reconcile(root, nil, nil, []string{"vendor"}); err != nil {
		t.Fatal(err)
	}
	active, err := s.GetAllCredentials()
	if err != nil || len(active) != 0 {
		t.Fatal("deleted eligible file was not retired", err)
	}
}

func TestReconcileRejectsOutOfScopeCoverage(t *testing.T) {
	root, outside := t.TempDir(), filepath.Join(t.TempDir(), "outside.txt")
	s, err := NewStore(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Reconcile(root, []string{outside}, []scanner.Match{{File: outside, MatchTxt: "synthetic-outside-fixture", LineNum: 1}}, nil); err == nil {
		t.Fatal("out-of-scope coverage accepted")
	}
	rows, err := s.GetAllCredentials()
	if err != nil || len(rows) != 0 {
		t.Fatal("invalid coverage changed evidence", err)
	}
}

func TestSymlinkAncestorRetainsEvidence(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skip("symlink privilege unavailable")
	}
	s, err := NewStore(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := filepath.Join(root, "linked", "missing.txt")
	if err := s.SaveMatch("synthetic-link-fixture", "Generic", 3, p, 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Reconcile(root, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	rows, err := s.GetAllCredentials()
	if err != nil || len(rows) != 1 {
		t.Fatal("symlink proved absence", err)
	}
}
