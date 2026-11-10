package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ScanRecord and Evidence contain metadata only; source text is never persisted.
type ScanRecord struct {
	ID                int64
	At, Root, Outcome string
	Files             int
	Events            []Evidence
}
type Evidence struct {
	Hash, FilePath string
	LineNumber     int
	State, Reason  string
}
type occurrence struct {
	hash, path string
	line       int
	active     bool
}

func snapshot(tx *sql.Tx) (map[int64]occurrence, error) {
	rows, err := tx.Query(`SELECT l.id,c.hash,l.file_path,l.line_number,l.active_status FROM locations l JOIN credentials c ON c.id=l.credential_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int64]occurrence{}
	for rows.Next() {
		var id int64
		var o occurrence
		if err := rows.Scan(&id, &o.hash, &o.path, &o.line, &o.active); err != nil {
			return nil, err
		}
		result[id] = o
	}
	return result, rows.Err()
}
func coverageReason(root, path string, files map[string]bool, ignored []string) string {
	if !filepath.IsAbs(path) {
		return "legacy relative location"
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "outside selected scope"
	}
	for _, part := range strings.Split(filepath.Dir(rel), string(os.PathSeparator)) {
		for _, name := range ignored {
			if part == name {
				return "excluded directory"
			}
		}
	}
	ancestor := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		ancestor = filepath.Join(ancestor, part)
		if info, e := os.Lstat(ancestor); e == nil && info.Mode()&os.ModeSymlink != 0 {
			return "symlink outside coverage"
		}
	}
	if files[path] {
		return "eligible text file checked"
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "location no longer exists"
	}
	return "not checked: binary, non-regular, or filesystem changed"
}
func appendTimeline(tx *sql.Tx, root string, files map[string]bool, ignored []string, before map[int64]occurrence) error {
	after, err := snapshot(tx)
	if err != nil {
		return err
	}
	result, err := tx.Exec(`INSERT INTO scan_history(at,root,outcome,files) VALUES(?,?,?,?)`, time.Now().UTC().Format(time.RFC3339Nano), root, "complete", len(files))
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	for key, o := range after {
		previous, exists := before[key]
		observed := files[o.path] && o.active
		if !observed && (!exists || !previous.active) {
			continue
		}
		state := "unverified"
		if observed {
			state = "new"
			if exists {
				state = "reappeared"
				if previous.active {
					state = "still present"
				}
			}
		} else if !o.active {
			state = "absent from checked coverage"
		}
		if _, err := tx.Exec(`INSERT INTO evidence_events(scan_id,hash,file_path,line_number,state,reason) VALUES(?,?,?,?,?,?)`, id, o.hash, o.path, o.line, state, coverageReason(root, o.path, files, ignored)); err != nil {
			return err
		}
	}
	return nil
}

// Failed attempts record no partial matches and never change occurrence evidence.
func (s *Store) RecordIncomplete(root, outcome string) error {
	if outcome != "failed" && outcome != "cancelled" {
		outcome = "failed"
	}
	_, err := s.db.Exec(`INSERT INTO scan_history(at,root,outcome,files) VALUES(?,?,?,0)`, time.Now().UTC().Format(time.RFC3339Nano), root, outcome)
	return err
}

// Timeline returns recent scans; hash filters events, never changes scan outcomes.
func (s *Store) Timeline(hash string) ([]ScanRecord, error) {
	rows, err := s.db.Query(`SELECT id,at,root,outcome,files FROM scan_history ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	result := []ScanRecord{}
	for rows.Next() {
		var r ScanRecord
		if err := rows.Scan(&r.ID, &r.At, &r.Root, &r.Outcome, &r.Files); err != nil {
			rows.Close()
			return nil, err
		}
		r.Events = []Evidence{}
		result = append(result, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range result {
		events, err := s.db.Query(`SELECT hash,file_path,line_number,state,reason FROM evidence_events WHERE scan_id=? AND (?='' OR hash=?) ORDER BY file_path,line_number,hash`, result[i].ID, hash, hash)
		if err != nil {
			return nil, err
		}
		for events.Next() {
			var e Evidence
			if err := events.Scan(&e.Hash, &e.FilePath, &e.LineNumber, &e.State, &e.Reason); err != nil {
				events.Close()
				return nil, err
			}
			result[i].Events = append(result[i].Events, e)
		}
		err = events.Err()
		events.Close()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
