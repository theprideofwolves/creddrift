// internal/ui/server.go
package ui

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/varkuru/creddrift/config"
	"github.com/varkuru/creddrift/internal/store"
	"github.com/varkuru/creddrift/scanner"
)

//go:embed index.html
var uiFiles embed.FS

// StartServer begins listening on the given port and serving the CredDrift dashboard
func StartServer(dbStore *store.Store, cfg *config.Config, s *scanner.SecretScanner, port string) {
	tmpl, err := template.ParseFS(uiFiles, "index.html")
	if err != nil {
		log.Fatalf("Failed to parse embedded UI templates: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		view := r.URL.Query().Get("view")

		var creds []store.Credential
		var fetchErr error

		if view == "ignored" {
			creds, fetchErr = dbStore.GetIgnoredCredentials()
		} else if view == "history" {
			creds, fetchErr = dbStore.GetRotationHistory()
		} else {
			creds, fetchErr = dbStore.GetAllCredentials()
		}

		if fetchErr != nil {
			http.Error(w, "Failed to fetch credentials", http.StatusInternalServerError)
			return
		}

		data := struct {
			Credentials []store.Credential
			View        string
		}{
			Credentials: creds,
			View:        view,
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Template execution error: %v", err)
		}
	})

	http.HandleFunc("/api/set-path", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			path := r.FormValue("path")
			if path != "" {
				cfg.ScanTargets = []string{path}
				matches, err := s.Scan()
				if err == nil {
					for _, m := range matches {
						_ = dbStore.SaveMatch(m.MatchTxt, m.Type, m.Entropy, m.File, m.LineNum)
					}
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/rotate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			for _, hash := range strings.Split(r.FormValue("hashes"), ",") {
				if hash != "" {
					_ = dbStore.MarkRotated(strings.TrimSpace(hash))
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/ignore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			for _, hash := range strings.Split(r.FormValue("hashes"), ",") {
				if hash != "" {
					_ = dbStore.IgnoreSecret(strings.TrimSpace(hash))
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/restore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			for _, hash := range strings.Split(r.FormValue("hashes"), ",") {
				if hash != "" {
					_ = dbStore.RestoreSecret(strings.TrimSpace(hash))
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/reveal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			paths := r.FormValue("paths")
			if paths != "" {
				parts := strings.Split(paths, ",")
				firstPath := strings.TrimSpace(parts[0])
				
				switch runtime.GOOS {
				case "windows":
					_ = exec.Command("explorer.exe", "/select,"+firstPath).Start()
				case "darwin":
					_ = exec.Command("open", "-R", firstPath).Start()
				case "linux":
					_ = exec.Command("xdg-open", filepath.Dir(firstPath)).Start()
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/browse", func(w http.ResponseWriter, r *http.Request) {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("powershell", "-NoProfile", "-Command", "Add-Type -AssemblyName System.windows.forms; $f = New-Object System.Windows.Forms.FolderBrowserDialog; if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $f.SelectedPath }")
		case "darwin":
			cmd = exec.Command("osascript", "-e", "POSIX path of (choose folder)")
		case "linux":
			cmd = exec.Command("zenity", "--file-selection", "--directory")
		default:
			http.Error(w, "Unsupported OS", http.StatusInternalServerError)
			return
		}

		out, err := cmd.Output()
		if err != nil {
			http.Error(w, "Failed to browse", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.TrimSpace(string(out))))
	})

	log.Printf("Web Dashboard starting on http://localhost%s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
