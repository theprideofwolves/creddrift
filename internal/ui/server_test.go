package ui

import (
	"context"
	"encoding/json"
	"github.com/theprideofwolves/creddrift/config"
	"github.com/theprideofwolves/creddrift/internal/store"
	"github.com/theprideofwolves/creddrift/internal/ticket"
	"github.com/theprideofwolves/creddrift/scanner"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOwnFolderScanFailureAndReconciliation(t *testing.T) {
	dir, source := t.TempDir(), t.TempDir()
	db, err := store.NewStore(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tickets, err := ticket.Open(filepath.Join(dir, "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	h := NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	post := func(form url.Values) string {
		t.Helper()
		r := httptest.NewRequest("POST", "http://localhost/?token=token", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if out.Code != 200 {
			t.Fatal(out.Code, out.Body.String())
		}
		return out.Body.String()
	}
	file := filepath.Join(source, "app.env")
	raw := "synthetic-own-folder-only"
	if err := os.WriteFile(file, []byte("password="+raw+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if page := post(url.Values{"action": {"scan"}, "path": {"."}}); !strings.Contains(page, "absolute directory") {
		t.Fatal("relative scan accepted")
	}
	if page := post(url.Values{"action": {"scan"}, "path": {filepath.Join(source, "missing")}}); !strings.Contains(page, "existing real directory") {
		t.Fatal("missing scan accepted")
	}
	if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); !os.IsNotExist(err) {
		t.Fatal("invalid scan marked complete", err)
	}
	rows, err := db.GetAllCredentials()
	if err != nil || len(rows) != 0 {
		t.Fatal("invalid request scanned source", rows, err)
	}
	if page := post(url.Values{"action": {"scan"}, "path": {source}}); !strings.Contains(page, "Complete:") || strings.Contains(page, raw) {
		t.Fatal("scan failed or raw value exposed", page)
	}
	rows, err = db.GetAllCredentials()
	if err != nil || len(rows) != 1 || rows[0].FilePath != file {
		t.Fatal(rows, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); err != nil {
		t.Fatal("successful scan progress not saved", err)
	}
	// A failed rescan must not replace earlier evidence or mark a fresh guide complete.
	if err := os.Remove(filepath.Join(dir, "onboarding.json")); err != nil {
		t.Fatal(err)
	}
	h = NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	if err := os.WriteFile(file, []byte(strings.Repeat("x", 2<<20)), 0600); err != nil {
		t.Fatal(err)
	}
	if page := post(url.Values{"action": {"scan"}, "path": {source}}); !strings.Contains(page, "Scan incomplete; existing findings retained") {
		t.Fatal(page)
	}
	rows, err = db.GetAllCredentials()
	if err != nil || len(rows) != 1 {
		t.Fatal("failed rescan lost findings", rows, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "onboarding.json")); !os.IsNotExist(err) {
		t.Fatal("failed scan marked complete", err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	post(url.Values{"action": {"scan"}, "path": {source}})
	rows, err = db.GetAllCredentials()
	if err != nil || len(rows) != 0 {
		t.Fatal("successful rescan did not retire removed location", rows, err)
	}
	retired, err := db.GetRetiredCredentials()
	if err != nil || len(retired) != 1 {
		t.Fatal(retired, err)
	}
	out := httptest.NewRecorder()
	h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/export?token=token&view=retired", nil))
	var exported []store.Credential
	if err := json.Unmarshal(out.Body.Bytes(), &exported); err != nil || len(exported) != 1 || strings.Contains(out.Body.String(), raw) {
		t.Fatal("invalid/redaction export", out.Body.String(), err)
	}
}

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
	r := httptest.NewRequest("POST", "http://localhost/?token=token", strings.NewReader("action=demo"))
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
	r = httptest.NewRequest("POST", "http://localhost/?token=token", strings.NewReader(url.Values{"action": {"ticket"}, "hash": {rows[0].Hash}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "Synthetic findings cannot") {
		t.Fatal(w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/export?token=token", nil))
	if strings.Contains(w.Body.String(), "synthetic-demo-only-abc123") {
		t.Fatal("raw export")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "db"))
	if strings.Contains(string(b), "synthetic-demo-only-abc123") {
		t.Fatal("raw candidate stored")
	}
}
func TestSetupFolderPrefillsWithoutScanning(t *testing.T) {
	dir := t.TempDir()
	source := t.TempDir()
	db, e := store.NewStore(filepath.Join(dir, "db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	tickets, _ := ticket.Open(filepath.Join(dir, "tickets.json"))
	cfg := &config.Config{}
	h := NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	post := func(path string, form url.Values) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://localhost"+path+"?token=token", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out
	}
	out := post("/setup/check", url.Values{"check": {"input"}, "path": {source}})
	if !strings.Contains(out.Body.String(), "Folder is readable") {
		t.Fatal(out.Body.String())
	}
	rows, _ := db.GetAllCredentials()
	if len(rows) != 0 {
		t.Fatal("setup scanned source")
	}
	post("/onboarding/start", url.Values{"mode": {"own"}})
	out = httptest.NewRecorder()
	h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/?token=token", nil))
	if !strings.Contains(out.Body.String(), source) {
		t.Fatal("selected folder not prefilled")
	}
}

func TestDemoIdentityCannotHandoffFromAnotherLocation(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewStore(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tickets, err := ticket.Open(filepath.Join(dir, "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Insert the external occurrence first so it is selected before the demo occurrence.
	if err := db.SaveMatch("synthetic-demo-only-abc123", "Generic", 4, filepath.Join(t.TempDir(), "copy.txt"), 1); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	h := NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	post := func(form url.Values) string {
		r := httptest.NewRequest("POST", "http://localhost/?token=token", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out.Body.String()
	}
	post(url.Values{"action": {"demo"}})
	page := post(url.Values{"action": {"ticket"}, "hash": {store.HashSecret("synthetic-demo-only-abc123")}})
	if !strings.Contains(page, "Synthetic findings cannot create real Jira tickets") {
		t.Fatal("demo identity was not blocked across locations")
	}
	if err := os.Remove(filepath.Join(dir, "synthetic-demo", "sample.txt")); err != nil {
		t.Fatal(err)
	}
	post(url.Values{"action": {"scan"}, "path": {filepath.Join(dir, "synthetic-demo")}})
	page = post(url.Values{"action": {"ticket"}, "hash": {store.HashSecret("synthetic-demo-only-abc123")}})
	if !strings.Contains(page, "Synthetic findings cannot create real Jira tickets") {
		t.Fatal("retired demo identity was not blocked")
	}
	if len(tickets.Records) != 0 {
		t.Fatal("demo created delivery intent")
	}
}

func TestCancelledRescanRetainsEvidence(t *testing.T) {
	dir, source := t.TempDir(), t.TempDir()
	db, err := store.NewStore(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	file := filepath.Join(source, "sample.txt")
	if err := os.WriteFile(file, []byte("password=synthetic-cancel-fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tickets, err := ticket.Open(filepath.Join(dir, "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	h := NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	scan := func(ctx context.Context) string {
		r := httptest.NewRequest("POST", "http://localhost/?token=token", strings.NewReader(url.Values{"action": {"scan"}, "path": {source}}.Encode())).WithContext(ctx)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Body.String()
	}
	scan(context.Background())
	before, err := db.GetAllCredentials()
	if err != nil || len(before) != 1 {
		t.Fatal("initial evidence missing", err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !strings.Contains(scan(ctx), "Scan incomplete; existing findings retained") {
		t.Fatal("cancelled request appeared successful")
	}
	after, err := db.GetAllCredentials()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("cancelled scan changed evidence", err)
	}
}

func TestViewExportLinksAndEmptyArray(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewStore(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tickets, err := ticket.Open(filepath.Join(dir, "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	h := NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	for _, view := range []string{"", "ignored", "retired", "history"} {
		out := httptest.NewRecorder()
		h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/export?token=token&view="+view, nil))
		if strings.TrimSpace(out.Body.String()) != "[]" {
			t.Fatal("empty export must be a JSON array")
		}
	}
	if err := db.SaveMatch("synthetic-export-fixture", "Generic", 3, filepath.Join(t.TempDir(), "missing.txt"), 1); err != nil {
		t.Fatal(err)
	}
	start := httptest.NewRequest("POST", "http://localhost/onboarding/start?token=token", strings.NewReader("mode=resume"))
	start.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), start)
	for _, view := range []string{"ignored", "retired", "history"} {
		out := httptest.NewRecorder()
		h.ServeHTTP(out, httptest.NewRequest("GET", "http://localhost/?token=token&view="+view, nil))
		if !strings.Contains(out.Body.String(), `/export?token=token&view=`+view) {
			t.Fatal("export link lost selected view")
		}
	}
}

func TestThemeAcrossWelcomeSetupAndWorkspace(t *testing.T) {
	dir := t.TempDir()
	db, err := store.NewStore(filepath.Join(dir, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tickets, err := ticket.Open(filepath.Join(dir, "tickets.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	h := NewHandler(db, cfg, scanner.NewScanner(cfg), tickets, dir, "token")
	r := httptest.NewRequest("POST", "http://localhost/theme?token=token", strings.NewReader("theme=dark"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 303 || len(w.Result().Cookies()) != 1 {
		t.Fatal("theme form failed")
	}
	for _, path := range []string{"/welcome", "/setup", "/"} {
		r := httptest.NewRequest("GET", "http://localhost"+path+"?token=token", nil)
		r.AddCookie(w.Result().Cookies()[0])
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if !strings.Contains(out.Body.String(), `data-theme="dark"`) || !strings.Contains(out.Body.String(), "Light mode") || strings.Contains(out.Body.String(), "ZgotmplZ") {
			t.Fatal("theme missing or template escaping failed")
		}
	}
	for _, tc := range []struct{ query, origin string }{{"", ""}, {"?token=token", "https://evil.test"}} {
		r := httptest.NewRequest("POST", "http://localhost/theme"+tc.query, strings.NewReader("theme=dark"))
		r.Header.Set("Origin", tc.origin)
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		if out.Code != 403 || len(out.Result().Cookies()) != 0 {
			t.Fatal("theme bypassed request guards")
		}
	}
}
