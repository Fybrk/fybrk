package core

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Fybrk represents the main application instance
type Fybrk struct {
	syncPath       string
	db             *sql.DB
	key            []byte
	syncEngine     *SyncEngine
	networkManager *NetworkManager
}

// Config holds configuration for Fybrk instance
type Config struct {
	SyncPath string
}

// PairData represents pairing information
type PairData struct {
	URL       string
	ExpiresAt time.Time
}

// New creates a new Fybrk instance with auto-initialization
func New(config Config) (*Fybrk, error) {
	if config.SyncPath == "" {
		return nil, fmt.Errorf("sync path is required")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(config.SyncPath)
	if err != nil {
		return nil, fmt.Errorf("invalid sync path: %w", err)
	}

	f := &Fybrk{
		syncPath: absPath,
	}

	// Auto-initialize
	if err := f.initialize(); err != nil {
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	// Initialize sync engine
	syncEngine, err := NewSyncEngine(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync engine: %w", err)
	}
	f.syncEngine = syncEngine

	// Initialize network manager
	f.networkManager = NewNetworkManager(f, syncEngine)

	return f, nil
}

// initialize sets up the sync directory and database
func (f *Fybrk) initialize() error {
	// Create sync directory if it doesn't exist
	if err := os.MkdirAll(f.syncPath, 0755); err != nil {
		return fmt.Errorf("failed to create sync directory: %w", err)
	}

	// Create .fybrk directory
	fybrDir := filepath.Join(f.syncPath, ".fybrk")
	if err := os.MkdirAll(fybrDir, 0755); err != nil {
		return fmt.Errorf("failed to create .fybrk directory: %w", err)
	}

	// Initialize or load encryption key
	if err := f.initializeKey(); err != nil {
		return fmt.Errorf("failed to initialize key: %w", err)
	}

	// Initialize database with robust error handling
	if err := f.initializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	return nil
}

// initializeKey creates or loads the encryption key
func (f *Fybrk) initializeKey() error {
	keyPath := filepath.Join(f.syncPath, ".fybrk", "key")

	// Try to load existing key
	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) == 32 {
			f.key = data
			return nil
		}
	}

	// Generate new key
	f.key = make([]byte, 32)
	if _, err := rand.Read(f.key); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Save key
	if err := os.WriteFile(keyPath, f.key, 0600); err != nil {
		return fmt.Errorf("failed to save key: %w", err)
	}

	return nil
}

// initializeDatabase sets up SQLite database with robust error handling
func (f *Fybrk) initializeDatabase() error {
	dbPath := filepath.Join(f.syncPath, ".fybrk", "metadata.db")

	// Remove any stale journal files that cause issues
	journalPath := dbPath + "-journal"
	if _, err := os.Stat(journalPath); err == nil {
		os.Remove(journalPath) // Ignore errors, best effort cleanup
	}

	// Open database with proper settings
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Create tables if they don't exist
	if err := f.createTables(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	f.db = db
	return nil
}

// createTables creates the necessary database tables
func (f *Fybrk) createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		size INTEGER NOT NULL,
		modified_at INTEGER NOT NULL,
		hash TEXT NOT NULL,
		created_at INTEGER DEFAULT (strftime('%s', 'now'))
	);
	
	CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
	CREATE INDEX IF NOT EXISTS idx_files_modified ON files(modified_at);
	`

	_, err := db.Exec(schema)
	return err
}

// GeneratePairData creates pairing information for other devices
func (f *Fybrk) GeneratePairData() (*PairData, error) {
	if f.key == nil {
		return nil, fmt.Errorf("encryption key not initialized")
	}

	// Create pairing URL with minimal data
	keyHex := hex.EncodeToString(f.key)
	expiresAt := time.Now().Add(10 * time.Minute)

	// Simple URL format - easy to copy/paste
	url := fmt.Sprintf("fybrk://pair?key=%s&path=%s&expires=%d",
		keyHex, f.syncPath, expiresAt.Unix())

	return &PairData{
		URL:       url,
		ExpiresAt: expiresAt,
	}, nil
}

// JoinFromPairData joins an existing sync from pairing URL
func JoinFromPairData(pairURL string, localPath string) (*Fybrk, error) {
	if !strings.HasPrefix(pairURL, "fybrk://pair?") {
		return nil, fmt.Errorf("invalid pairing URL format")
	}

	// TODO: Parse URL and extract key, remote path, etc.
	// For MVP, create local instance and return

	if localPath == "" {
		// Default to current directory
		localPath = "."
	}

	config := Config{SyncPath: localPath}
	return New(config)
}

// IsValidPairURL checks if a string is a valid pairing URL
func IsValidPairURL(s string) bool {
	return strings.HasPrefix(s, "fybrk://pair?")
}

// Close cleanly shuts down the Fybrk instance
func (f *Fybrk) Close() error {
	if f.syncEngine != nil {
		f.syncEngine.Stop()
	}
	if f.networkManager != nil {
		f.networkManager.Stop()
	}
	if f.db != nil {
		return f.db.Close()
	}
	return nil
}

// GetSyncPath returns the current sync path
func (f *Fybrk) GetSyncPath() string {
	return f.syncPath
}

// GetKey returns the encryption key (for testing)
func (f *Fybrk) GetKey() []byte {
	return f.key
}

// FileRecord represents a file in the database
type FileRecord struct {
	ID         int64
	Path       string
	Size       int64
	ModifiedAt time.Time
	Hash       string
	CreatedAt  time.Time
}

// updateFileRecord updates or inserts a file record
func (f *Fybrk) updateFileRecord(path string, size int64, modTime time.Time, hash string) error {
	query := `
		INSERT OR REPLACE INTO files (path, size, modified_at, hash)
		VALUES (?, ?, ?, ?)
	`
	_, err := f.db.Exec(query, path, size, modTime.Unix(), hash)
	return err
}

// getFileRecord retrieves a file record
func (f *Fybrk) getFileRecord(path string) (*FileRecord, error) {
	query := `
		SELECT id, path, size, modified_at, hash, created_at
		FROM files WHERE path = ?
	`
	row := f.db.QueryRow(query, path)

	var record FileRecord
	var modifiedAt, createdAt int64

	err := row.Scan(&record.ID, &record.Path, &record.Size, &modifiedAt, &record.Hash, &createdAt)
	if err != nil {
		return nil, err
	}

	record.ModifiedAt = time.Unix(modifiedAt, 0)
	record.CreatedAt = time.Unix(createdAt, 0)

	return &record, nil
}

// deleteFileRecord removes a file record
func (f *Fybrk) deleteFileRecord(path string) error {
	query := `DELETE FROM files WHERE path = ?`
	_, err := f.db.Exec(query, path)
	return err
}

// StartSync starts the sync engine and network server
func (f *Fybrk) StartSync() error {
	if err := f.networkManager.StartServer(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	if err := f.syncEngine.Start(); err != nil {
		return fmt.Errorf("failed to start sync engine: %w", err)
	}

	return nil
}

// GetServerAddress returns the local server address
func (f *Fybrk) GetServerAddress() string {
	return f.networkManager.GetLocalAddress()
}

// ConnectToPeer connects to a remote peer
func (f *Fybrk) ConnectToPeer(address string) error {
	return f.networkManager.ConnectToPeer(address)
}
