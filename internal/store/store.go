package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
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

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &Store{db: db}
	if err := store.createTables(); err != nil {
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
	);`

	_, err := s.db.Exec(query)
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
	INSERT OR IGNORE INTO locations (credential_id, file_path, line_number, active_status)
	VALUES (?, ?, ?, ?);`

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
}

// baseSelect defines the common fields required by runQuery.
const baseSelect = `
	SELECT c.hash, c.secret_type, c.entropy_score, c.last_rotated, l.file_path, l.line_number, c.last_seen,
	(SELECT COUNT(DISTINCT file_path) FROM locations WHERE credential_id = c.id),
	(l.id > (SELECT MIN(id) FROM locations WHERE credential_id = c.id))
`

// GetAllCredentials fetches active secrets.
func (s *Store) GetAllCredentials() ([]Credential, error) {
	query := baseSelect + `
	FROM credentials c
	JOIN locations l ON c.id = l.credential_id
	LEFT JOIN ignored_secrets i ON c.hash = i.hash
	WHERE i.hash IS NULL
	ORDER BY c.last_seen DESC`
	return s.runQuery(query)
}

// GetIgnoredCredentials fetches only secrets marked as ignored.
func (s *Store) GetIgnoredCredentials() ([]Credential, error) {
	query := baseSelect + `
	FROM credentials c
	JOIN locations l ON c.id = l.credential_id
	JOIN ignored_secrets i ON c.hash = i.hash
	ORDER BY c.last_seen DESC`
	return s.runQuery(query)
}

// GetRotationHistory fetches active secrets ordered by most recent rotation.
func (s *Store) GetRotationHistory() ([]Credential, error) {
	query := baseSelect + `
	FROM credentials c
	JOIN locations l ON c.id = l.credential_id
	LEFT JOIN ignored_secrets i ON c.hash = i.hash
	WHERE i.hash IS NULL AND c.last_rotated IS NOT NULL AND c.last_rotated != c.first_seen
	ORDER BY c.last_rotated DESC`
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
	now := time.Now()

	for rows.Next() {
		var c Credential
		var lastRotated, lastSeen time.Time
		var isDrift int

		if err := rows.Scan(&c.Hash, &c.SecretType, &c.EntropyScore, &lastRotated, &c.FilePath, &c.LineNumber, &lastSeen, &c.BlastRadius, &isDrift); err != nil {
			return nil, err
		}

		if now.Sub(lastSeen).Minutes() > 5 {
			continue // Hide stale/removed secrets automatically
		}

		c.LastRotated = lastRotated.Format("2006-01-02 15:04")

		c.Status = "Active"
		if isDrift > 0 {
			c.Status = "Drift Detected"
		} else if now.Sub(lastRotated).Hours() > 24*90 {
			c.Status = "Needs Rotation"
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
	_, err := s.db.Exec("UPDATE credentials SET last_rotated = ? WHERE hash = ?", time.Now(), hash)
	return err
}

// RestoreSecret deletes a hash from the ignored_secrets table so it shows up again.
func (s *Store) RestoreSecret(hash string) error {
	_, err := s.db.Exec("DELETE FROM ignored_secrets WHERE hash = ?", hash)
	return err
}
