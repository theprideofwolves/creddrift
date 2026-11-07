package store

import (
	"path/filepath"
	"testing"
)

func TestFindingLifecycle(t *testing.T) {
	s, err := NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, path := range []string{"a.txt", "b.txt", "a.txt"} {
		if err := s.SaveMatch("synthetic-password", "Generic", 4, path, 1); err != nil {
			t.Fatal(err)
		}
	}
	hash := HashSecret("synthetic-password")
	rows, err := s.GetAllCredentials()
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if rows[0].BlastRadius != 2 {
		t.Fatal("location count")
	}
	if _, err := s.db.Exec("UPDATE credentials SET last_seen = datetime('now','-1 hour')"); err != nil {
		t.Fatal(err)
	}
	rows, err = s.GetAllCredentials()
	if err != nil || len(rows) != 2 {
		t.Fatalf("findings disappeared: %v", err)
	}
	if err := s.IgnoreSecret(hash); err != nil {
		t.Fatal(err)
	}
	rows, err = s.GetAllCredentials()
	if err != nil || len(rows) != 0 {
		t.Fatal("ignore failed")
	}
	if err := s.RestoreSecret(hash); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRotated(hash); err != nil {
		t.Fatal(err)
	}
	rows, err = s.GetRotationHistory()
	if err != nil || len(rows) != 2 {
		t.Fatalf("rotation history: %+v %v", rows, err)
	}
}
