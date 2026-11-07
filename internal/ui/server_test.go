package ui

import (
	"github.com/theprideofwolves/creddrift/config"
	"github.com/theprideofwolves/creddrift/internal/store"
	"github.com/theprideofwolves/creddrift/internal/ticket"
	"github.com/theprideofwolves/creddrift/scanner"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoAndRedaction(t *testing.T) {
	dir := t.TempDir()
	db, e := store.NewStore(filepath.Join(dir, "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	tickets, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	cfg := &config.Config{}
	h := NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	r := httptest.NewRequest("POST", "http://localhost/", strings.NewReader("action=demo"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || strings.Contains(w.Body.String(), "synthetic-demo-only-abc123") {
		t.Fatal("raw candidate exposed")
	}
	rows, _ := db.GetAllCredentials()
	if len(rows) != 1 {
		t.Fatal(rows)
	}
	r = httptest.NewRequest("POST", "http://localhost/", strings.NewReader(url.Values{"action": {"ticket"}, "hash": {rows[0].Hash}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "Synthetic findings cannot") {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export", nil))
	if strings.Contains(w.Body.String(), "synthetic-demo-only-abc123") {
		t.Fatal("raw export")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "db"))
	if strings.Contains(string(b), "synthetic-demo-only-abc123") {
		t.Fatal("raw candidate stored")
	}
}
