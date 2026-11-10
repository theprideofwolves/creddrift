package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/theprideofwolves/creddrift/scanner"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the sql.DB connection
type Store struct {
	db *sql.DB
}

// NewStore initializes the database, creating the file if it doesn't exist.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &Store{db: db}
	if err := store.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

// createTables initializes the SQLite schema based on spec.md
func (s *Store) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hash TEXT UNIQUE NOT NULL,
		secret_type TEXT NOT NULL,
		entropy_score REAL NOT NULL,
		first_seen DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		last_rotated DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS locations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		credential_id INTEGER NOT NULL,
		file_path TEXT NOT NULL,
		line_number INTEGER NOT NULL,
		active_status BOOLEAN NOT NULL,
		FOREIGN KEY(credential_id) REFERENCES credentials(id),
		UNIQUE(credential_id, file_path, line_number)
	);
	
	CREATE TABLE IF NOT EXISTS ignored_secrets (
		hash TEXT PRIMARY KEY
	);
 CREATE TABLE IF NOT EXISTS scan_history(id INTEGER PRIMARY KEY,at TEXT NOT NULL,root TEXT NOT NULL,outcome TEXT NOT NULL,files INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS evidence_events(scan_id INTEGER NOT NULL,hash TEXT NOT NULL,file_path TEXT NOT NULL,line_number INTEGER NOT NULL,state TEXT NOT NULL,reason TEXT NOT NULL);
 CREATE INDEX IF NOT EXISTS evidence_scan ON evidence_events(scan_id);
`

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	rows, err := s.db.Query("PRAGMA table_info(credentials)")
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var def any
		if err = rows.Scan(&cid, &name, &typ, &notnull, &def, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == "rotation_recorded_at" {
			found = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !found {
		_, err = s.db.Exec("ALTER TABLE credentials ADD COLUMN rotation_recorded_at DATETIME")
	}
	return err
}

// HashSecret takes a raw string and returns its SHA 256 hash representation.
func HashSecret(raw string) string {
	h := sha256.New()
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}

// SaveMatch hashes a raw secret, and stores or updates its record and location in the database.
func (s *Store) SaveMatch(rawSecret, secretType string, entropy float64, filePath string, lineNum int) error {
	hash := HashSecret(rawSecret)
	now := time.Now()

	// 1. Insert new credential or update 'last_seen' if it already exists
	credQuery := `
	INSERT INTO credentials (hash, secret_type, entropy_score, first_seen, last_seen, last_rotated)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen;`

	_, err := s.db.Exec(credQuery, hash, secretType, entropy, now, now, now)
	if err != nil {
		return fmt.Errorf("failed to insert/update credential: %w", err)
	}

	// 2. Retrieve the credential ID (we do a SELECT because LastInsertId won't work on UPDATE conflict)
	var credID int64
	err = s.db.QueryRow("SELECT id FROM credentials WHERE hash = ?", hash).Scan(&credID)
	if err != nil {
		return fmt.Errorf("failed to retrieve credential ID: %w", err)
	}

	// 3. Insert location if it is a new occurrence
	locQuery := `
	INSERT INTO locations (credential_id, file_path, line_number, active_status)
	VALUES (?, ?, ?, ?) ON CONFLICT(credential_id,file_path,line_number) DO UPDATE SET active_status=1;`

	_, err = s.db.Exec(locQuery, credID, filePath, lineNum, true)
	if err != nil {
		return fmt.Errorf("failed to insert location: %w", err)
	}

	return nil
}

// Close gracefully closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Credential models the returned credential data for the UI
type Credential struct {
	Hash         string
	SecretType   string
	EntropyScore float64
	Status       string
	FilePath     string
	LineNumber   int
	BlastRadius  int
	LastRotated  string
	LastSeen     string
}

// baseSelect defines the common fields required by runQuery.
const baseSelect = `
	SELECT c.hash, c.secret_type, c.entropy_score, c.rotation_recorded_at, l.file_path, l.line_number, c.last_seen,
	(SELECT COUNT(DISTINCT file_path) FROM locations WHERE credential_id = c.id AND active_status=1),
	COALESCE((l.id > (SELECT MIN(id) FROM locations WHERE credential_id = c.id AND active_status=1)), 0)
`

// GetAllCredentials fetches active secrets.
func (s *Store) GetAllCredentials() ([]Credential, error) {
	query := baseSelect + `
	FROM credentials c
	JOIN locations l ON c.id = l.credential_id
	LEFT JOIN ignored_secrets i ON c.hash = i.hash
	WHERE i.hash IS NULL AND l.active_status=1
	ORDER BY c.last_seen DESC`
	return s.runQuery(query)
}

// GetIgnoredCredentials fetches only secrets marked as ignored.
func (s *Store) GetIgnoredCredentials() ([]Credential, error) {
	query := baseSelect + `
	FROM credentials c
	JOIN locations l ON c.id = l.credential_id
	JOIN ignored_secrets i ON c.hash = i.hash
 WHERE l.active_status=1
	ORDER BY c.last_seen DESC`
	return s.runQuery(query)
}

// GetRotationHistory fetches active secrets ordered by most recent rotation.
func (s *Store) GetRotationHistory() ([]Credential, error) {
	query := baseSelect + `
	FROM credentials c
	JOIN locations l ON c.id = l.credential_id
	LEFT JOIN ignored_secrets i ON c.hash = i.hash
	WHERE i.hash IS NULL AND l.active_status=1 AND c.rotation_recorded_at IS NOT NULL
	ORDER BY c.rotation_recorded_at DESC`
	return s.runQuery(query)
}

// runQuery executes the query and bundles the Credential objects
func (s *Store) runQuery(query string) ([]Credential, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var c Credential
		var lastRotated sql.NullTime
		var lastSeen time.Time
		var isDrift int

		if err := rows.Scan(&c.Hash, &c.SecretType, &c.EntropyScore, &lastRotated, &c.FilePath, &c.LineNumber, &lastSeen, &c.BlastRadius, &isDrift); err != nil {
			return nil, err
		}

		c.LastSeen = lastSeen.UTC().Format(time.RFC3339)
		c.LastRotated = "Not recorded"
		if lastRotated.Valid {
			c.LastRotated = lastRotated.Time.UTC().Format(time.RFC3339)
		}

		c.Status = "Potential secret"
		if isDrift > 0 {
			c.Status = "Additional recorded location"

		}

		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// IgnoreSecret adds a hash to the ignored_secrets table.
func (s *Store) IgnoreSecret(hash string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO ignored_secrets (hash) VALUES (?)", hash)
	return err
}

// MarkRotated updates the last_rotated time for a secret.
func (s *Store) MarkRotated(hash string) error {
	_, err := s.db.Exec("UPDATE credentials SET rotation_recorded_at = ? WHERE hash = ?", time.Now(), hash)
	return err
}

// RestoreSecret deletes a hash from the ignored_secrets table so it shows up again.
func (s *Store) RestoreSecret(hash string) error {
	_, err := s.db.Exec("DELETE FROM ignored_secrets WHERE hash = ?", hash)
	return err
}

// Reconcile only after a complete scan. Ignored/binary/unreadable files never prove absence.
func (s *Store) Reconcile(root string, scanned []string, matches []scanner.Match, ignored []string) error {
	root, e := filepath.Abs(root)
	if e != nil {
		return e
	}
	files := map[string]bool{}
	for _, p := range scanned {
		p, e = filepath.Abs(p)
		if e != nil {
			return e
		}
		rel, err := filepath.Rel(root, p)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("completed scan coverage outside selected root")
		}
		files[p] = true
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	before, e := snapshot(tx)
	if e != nil {
		return e
	}
	rows, e := tx.Query("SELECT id,file_path FROM locations WHERE active_status=1")
	if e != nil {
		return e
	}
	var deactivate []int
	for rows.Next() {
		var id int
		var path string
		if e = rows.Scan(&id, &path); e != nil {
			rows.Close()
			return e
		}
		if !filepath.IsAbs(path) {
			continue
		} // Legacy relative locations have no trustworthy original root.
		abs, e := filepath.Abs(path)
		if e != nil {
			rows.Close()
			return e
		}
		rel, e := filepath.Rel(root, abs)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		skip := false
		for _, component := range strings.Split(filepath.Dir(rel), string(os.PathSeparator)) {
			for _, name := range ignored {
				if component == name {
					skip = true
				}
			}
		}
		if skip {
			continue
		}
		ancestor := root
		for _, component := range strings.Split(rel, string(os.PathSeparator)) {
			ancestor = filepath.Join(ancestor, component)
			if info, err := os.Lstat(ancestor); err == nil && info.Mode()&os.ModeSymlink != 0 {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if files[abs] {
			deactivate = append(deactivate, id)
		} else if _, e = os.Stat(abs); os.IsNotExist(e) {
			deactivate = append(deactivate, id)
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range deactivate {
		if _, e = tx.Exec("UPDATE locations SET active_status=0 WHERE id=?", id); e != nil {
			return e
		}
	}
	now := time.Now().UTC()
	for _, m := range matches {
		path, e := filepath.Abs(m.File)
		if e != nil {
			return e
		}
		if !files[path] {
			return fmt.Errorf("candidate outside completed scan coverage")
		}
		hash := HashSecret(m.MatchTxt)
		if _, e = tx.Exec(`INSERT INTO credentials(hash,secret_type,entropy_score,first_seen,last_seen,last_rotated) VALUES(?,?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen`, hash, m.Type, m.Entropy, now, now, now); e != nil {
			return e
		}
		if _, e = tx.Exec(`INSERT INTO locations(credential_id,file_path,line_number,active_status) SELECT id,?,?,1 FROM credentials WHERE hash=? ON CONFLICT(credential_id,file_path,line_number) DO UPDATE SET active_status=1`, path, m.LineNum, hash); e != nil {
			return e
		}
	}
	if e := appendTimeline(tx, root, files, ignored, before); e != nil {
		return e
	}
	return tx.Commit()
}

func (s *Store) GetRetiredCredentials() ([]Credential, error) {
	query := baseSelect + ` FROM credentials c JOIN locations l ON c.id=l.credential_id WHERE l.active_status=0 ORDER BY c.last_seen DESC`
	rows, e := s.runQuery(query)
	for i := range rows {
		rows[i].Status = "Not observed in successful rescan (not provider revocation)"
	}
	return rows, e
}
