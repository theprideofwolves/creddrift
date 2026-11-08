package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/theprideofwolves/creddrift/config"
	"github.com/theprideofwolves/creddrift/internal/desktop"
	"github.com/theprideofwolves/creddrift/internal/onboarding"
	"github.com/theprideofwolves/creddrift/internal/store"
	"github.com/theprideofwolves/creddrift/internal/ticket"
	"github.com/theprideofwolves/creddrift/scanner"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func Serve(db *store.Store, cfg *config.Config, s *scanner.SecretScanner, dir string) error {
	tickets, e := ticket.Open(filepath.Join(dir, "tickets.json"))
	if e != nil {
		return e
	}
	return desktop.Serve("CredDrift", dir, func(token string) http.Handler { return NewHandler(db, cfg, s, tickets, dir, token) })
}
func NewHandler(db *store.Store, cfg *config.Config, s *scanner.SecretScanner, tickets *ticket.Store, dir, token string) http.Handler {
	guide := onboarding.New(onboarding.Config{Name: "CredDrift", Token: token, Dir: dir, Jira: true, Folder: true, Intro: "Find and review potential hardcoded credentials without exposing their values.", Expected: "The demo creates one clearly synthetic candidate in a separate sample folder. Its value stays hidden, and it cannot create real Jira tickets.", Next: "Inspect the finding location, export your review, or choose your own folder through setup. Nothing rotates a provider credential automatically.", Steps: []string{"Choose a source folder you are authorized to review. The check below only verifies that it is readable.", "Open the workspace and run a scan. Review potential candidates before treating them as real credentials.", "Local review needs no external account. Jira is optional, and its token is never saved."}, DemoFields: map[string]string{"action": "demo"}, Fields: []onboarding.Field{{Name: "path", Label: "Folder to review", Hint: "Choose folder, or paste an absolute directory path. Files are not uploaded."}}, Check: onboarding.CheckFolder})
	var mu sync.Mutex
	message := "Choose a folder through setup or paste it below, then run a scan. No scan runs automatically."
	demo := false
	tmpl := template.Must(template.New("page").Funcs(template.FuncMap{"onboarding": guide.Panel, "onboardingPath": func() string { return guide.Value("path") }}).Parse(strings.ReplaceAll(page, "STYLE", desktop.Style)))
	return guide.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path != "/" && r.URL.Path != "/export" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/export" && r.Method != "GET" {
			http.Error(w, "GET required", 405)
			return
		}
		if r.Method == "POST" {
			r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
			if e := r.ParseForm(); e != nil {
				http.Error(w, "Invalid request", 400)
				return
			}
			if len(r.PostForm.Encode()) > 64<<10 {
				http.Error(w, "Request too large", 400)
				return
			}
			action := r.FormValue("action")
			switch action {
			case "scan", "demo":
				root := strings.TrimSpace(r.FormValue("path"))
				demo = action == "demo"
				if demo {
					root = filepath.Join(dir, "synthetic-demo")
					if e := os.MkdirAll(root, 0700); e != nil {
						message = e.Error()
						break
					}
					if e := os.WriteFile(filepath.Join(root, "sample.txt"), []byte("password = synthetic-demo-only-abc123\n"), 0600); e != nil {
						message = e.Error()
						break
					}
				}
				abs, e := filepath.Abs(root)
				if e != nil || !filepath.IsAbs(root) {
					message = "Select an absolute directory path first."
					break
				}
				info, e := os.Lstat(abs)
				if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					message = "Choose an existing real directory, not a symlink."
					break
				}
				cfg.ScanTargets = []string{abs}
				ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
				matches, e := s.ScanContext(ctx)
				cancel()
				if e != nil {
					message = "Scan incomplete; existing findings retained. Check unreadable files, 1 MiB line limit, 32 MiB file limit, and permissions. " + e.Error()
					break
				}
				if e = db.Reconcile(abs, s.ScannedFiles, matches, cfg.IgnoreDirs); e != nil {
					message = "Could not reconcile scan; transaction rolled back."
					break
				}
				guide.Complete()
				message = fmt.Sprintf("Complete: %d eligible text files, %d potential matches. Binary files, symlinks and ignored directories are outside coverage.", len(s.ScannedFiles), len(matches))
			case "ignore", "restore", "rotate", "ticket":
				hash := r.FormValue("hash")
				all, e := db.GetAllCredentials()
				if e != nil {
					message = "Database unavailable"
					break
				}
				ignored, e := db.GetIgnoredCredentials()
				if e != nil {
					message = "Database unavailable"
					break
				}
				all = append(all, ignored...)
				var selected *store.Credential
				for i := range all {
					if all[i].Hash == hash {
						selected = &all[i]
						break
					}
				}
				if selected == nil {
					message = "Finding not found in active observations."
					break
				}
				switch action {
				case "ignore":
					e = db.IgnoreSecret(hash)
				case "restore":
					e = db.RestoreSecret(hash)
				case "rotate":
					e = db.MarkRotated(hash)
				case "ticket":
					demoPath := filepath.Join(dir, "synthetic-demo")
					rel, err := filepath.Rel(demoPath, selected.FilePath)
					isDemo := err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
					if isDemo {
						message = "Synthetic findings cannot create real Jira tickets."
						break
					}
					body := fmt.Sprintf("Potential hardcoded credential candidate. Type: %s. Local finding ID: %s. Local recorded file count: %d. Reviewed at %s. Investigate and revoke/rotate through the provider if confirmed. No credential values, source snippets, or local file paths are included. This app does not validate or rotate provider credentials.", selected.SecretType, hash, selected.BlastRadius, time.Now().UTC().Format(time.RFC3339))
					message, e = tickets.Handoff(r.Context(), ticket.Config{Origin: r.FormValue("jira"), Email: r.FormValue("email"), Token: r.FormValue("secret"), Project: r.FormValue("project"), Kind: r.FormValue("kind")}, "creddrift|"+hash, "Review potential hardcoded credential", body, r.FormValue("link"))
				}
				if e != nil {
					message = e.Error()
				} else if action != "ticket" {
					message = "Review action recorded. Provider credentials were not changed."
				}
			default:
				message = "Unsupported action"
			}
		}
		creds, e := db.GetAllCredentials()
		if r.URL.Query().Get("view") == "ignored" {
			creds, e = db.GetIgnoredCredentials()
		}
		if r.URL.Query().Get("view") == "retired" {
			creds, e = db.GetRetiredCredentials()
		}
		if r.URL.Query().Get("view") == "history" {
			creds, e = db.GetRotationHistory()
		}
		if e != nil {
			http.Error(w, "Cannot read findings", 500)
			return
		}
		if r.URL.Path == "/export" {
			if r.Method != "GET" {
				http.Error(w, "GET required", 405)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename=creddrift-findings.json")
			json.NewEncoder(w).Encode(creds)
			return
		}
		tmpl.Execute(w, struct {
			Token, Message string
			Credentials    []store.Credential
			Tickets        map[string]ticket.Record
			Demo           bool
		}{token, message, creds, tickets.Records, demo})
	}))
}

const page = `<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>CredDrift</title><style>STYLE</style><h1>CredDrift</h1>{{onboarding}}<p>Local credential discovery. Reviewed findings. Trustworthy rescans.</p><p class="notice">Finds potential credentials, not confirmed active keys. Entropy is a character statistic, not proof of password strength or validity. Raw candidates never appear in the UI or exports. Local file paths and hashes are sensitive; exports remain your responsibility.</p><p role="status">{{.Message}}</p><section><h2>Scan a directory</h2><form method="post" action="/?token={{.Token}}"><label>Absolute directory path<input name="path" value="{{onboardingPath}}" placeholder="C:\projects\example"></label><button name="action" value="scan">Scan and reconcile</button><button formaction="/?token={{.Token}}&onboarding=demo" name="action" value="demo">Run synthetic demo</button></form><p>Successful scans retire removed/moved occurrences only inside verified coverage. Failed scans retain earlier findings. Ignoring a candidate applies across all its locations.</p></section><nav><a href="/?token={{.Token}}">Active</a> | <a href="/?token={{.Token}}&view=ignored">Ignored</a> | <a href="/?token={{.Token}}&view=retired">Retired observations</a> | <a href="/?token={{.Token}}&view=history">User-recorded rotation actions</a> | <a href="/export?token={{.Token}}">Export active findings</a></nav><section id="results"><h2>Review findings</h2>{{range .Credentials}}<article><h3>{{.SecretType}} | {{.Status}}</h3><p>{{.FilePath}}:{{.LineNumber}}</p><small>ID: {{.Hash}}<br>Last observed: {{.LastSeen}}<br>Recorded files: {{.BlastRadius}} | Manual rotation action: {{.LastRotated}}</small><form method="post" action="/?token={{$.Token}}"><input type="hidden" name="hash" value="{{.Hash}}"><button name="action" value="ignore">Ignore candidate</button><button name="action" value="restore">Restore candidate</button><button name="action" value="rotate">Record my manual rotation action</button></form><details><summary>Review Jira disclosure and send</summary><form method="post" action="/?token={{$.Token}}"><input type="hidden" name="hash" value="{{.Hash}}"><p>Sends candidate type, ID, local file count and review time only. No raw credential, source snippet or path. API token is not saved. An uncertain create must be reconciled in Jira before another create.</p><section><h2>Jira handoff</h2><p>Review the selected record before sending. Sends only the disclosed candidate summary and correlation ID. Tokens are used for this request and never saved. Creates once; linked issues receive evidence comments. No automatic ticket closure. Inspect Jira before retrying an uncertain comment.</p><label>Jira origin<input name="jira" placeholder="https://team.atlassian.net" required></label><label>Email<input name="email" type="email" required autocomplete="off"></label><label>API token<input name="secret" type="password" required autocomplete="off"></label><label>Project<input name="project" placeholder="SEC"></label><label>Issue type<input name="kind" value="Task"></label><label>Existing key (optional; validates and links instead of creating)<input name="link" placeholder="SEC-123"></label><button name="action" value="ticket">Send reviewed record / link issue</button></section></form></details><hr></article>{{else}}<p>No findings in this view.</p>{{end}}</section><section><h2>Ticket delivery history</h2>{{range $id,$v:=.Tickets}}<p><small>{{$id}}: {{$v.Key}} ({{$v.State}})</small></p>{{else}}<p>No deliveries.</p>{{end}}</section><small>Single-user localhost tool. Legacy first-discovery timestamps are not presented as rotation. Close with Ctrl+C in the launcher. Findings and database backups must be protected.</small></html>`
