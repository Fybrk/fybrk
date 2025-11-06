package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew_ValidPath(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	if fybrk.GetSyncPath() != tempDir {
		t.Errorf("Expected sync path %s, got %s", tempDir, fybrk.GetSyncPath())
	}
}

func TestNew_EmptyPath(t *testing.T) {
	config := Config{SyncPath: ""}
	_, err := New(config)

	if err == nil {
		t.Fatal("Expected error for empty sync path")
	}

	if !strings.Contains(err.Error(), "sync path is required") {
		t.Errorf("Expected 'sync path is required' error, got: %v", err)
	}
}

func TestNew_RelativePath(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)

	os.Chdir(tempDir)

	config := Config{SyncPath: "."}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	// Resolve both paths to handle symlinks (like /var -> /private/var on macOS)
	expectedPath, _ := filepath.EvalSymlinks(tempDir)
	actualPath, _ := filepath.EvalSymlinks(fybrk.GetSyncPath())

	if actualPath != expectedPath {
		t.Errorf("Expected absolute path %s, got %s", expectedPath, actualPath)
	}
}

func TestNew_NonExistentPath(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent", "path")

	config := Config{SyncPath: nonExistentPath}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error (should auto-create), got: %v", err)
	}
	defer fybrk.Close()

	// Verify directory was created
	if _, err := os.Stat(nonExistentPath); os.IsNotExist(err) {
		t.Error("Expected directory to be auto-created")
	}
}

func TestInitialize_CreatesDirectories(t *testing.T) {
	tempDir := t.TempDir()
	syncPath := filepath.Join(tempDir, "sync")

	config := Config{SyncPath: syncPath}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	// Check sync directory exists
	if _, err := os.Stat(syncPath); os.IsNotExist(err) {
		t.Error("Sync directory was not created")
	}

	// Check .fybrk directory exists
	fybrDir := filepath.Join(syncPath, ".fybrk")
	if _, err := os.Stat(fybrDir); os.IsNotExist(err) {
		t.Error(".fybrk directory was not created")
	}
}

func TestInitialize_CreatesKey(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	// Check key was generated
	key := fybrk.GetKey()
	if len(key) != 32 {
		t.Errorf("Expected 32-byte key, got %d bytes", len(key))
	}

	// Check key file exists
	keyPath := filepath.Join(tempDir, ".fybrk", "key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Key file was not created")
	}
}

func TestInitialize_LoadsExistingKey(t *testing.T) {
	tempDir := t.TempDir()
	fybrDir := filepath.Join(tempDir, ".fybrk")
	os.MkdirAll(fybrDir, 0755)

	// Create existing key
	existingKey := make([]byte, 32)
	for i := range existingKey {
		existingKey[i] = byte(i)
	}
	keyPath := filepath.Join(fybrDir, "key")
	os.WriteFile(keyPath, existingKey, 0600)

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	// Check key was loaded correctly
	key := fybrk.GetKey()
	if len(key) != 32 {
		t.Errorf("Expected 32-byte key, got %d bytes", len(key))
	}

	for i, b := range key {
		if b != byte(i) {
			t.Errorf("Key mismatch at position %d: expected %d, got %d", i, i, b)
		}
	}
}

func TestInitialize_CreatesDatabase(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	// Check database file exists
	dbPath := filepath.Join(tempDir, ".fybrk", "metadata.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Check database is accessible
	if fybrk.db == nil {
		t.Error("Database connection is nil")
	}

	// Test database connection
	if err := fybrk.db.Ping(); err != nil {
		t.Errorf("Database ping failed: %v", err)
	}
}

func TestInitialize_HandlesCorruptedJournal(t *testing.T) {
	tempDir := t.TempDir()
	fybrDir := filepath.Join(tempDir, ".fybrk")
	os.MkdirAll(fybrDir, 0755)

	// Create a corrupted journal file
	journalPath := filepath.Join(fybrDir, "metadata.db-journal")
	os.WriteFile(journalPath, []byte("corrupted data"), 0644)

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error (should handle corrupted journal), got: %v", err)
	}
	defer fybrk.Close()

	// Journal file should be cleaned up
	if _, err := os.Stat(journalPath); err == nil {
		t.Error("Corrupted journal file was not cleaned up")
	}
}

func TestGeneratePairData_ValidData(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	pairData, err := fybrk.GeneratePairData()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pairData == nil {
		t.Fatal("Expected pair data, got nil")
	}

	if !strings.HasPrefix(pairData.URL, "fybrk://pair?") {
		t.Errorf("Expected URL to start with 'fybrk://pair?', got: %s", pairData.URL)
	}

	if pairData.ExpiresAt.Before(time.Now()) {
		t.Error("Expected expiration time to be in the future")
	}

	if pairData.ExpiresAt.After(time.Now().Add(11 * time.Minute)) {
		t.Error("Expected expiration time to be within 10 minutes")
	}
}

func TestGeneratePairData_ContainsKey(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	pairData, err := fybrk.GeneratePairData()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(pairData.URL, "key=") {
		t.Error("Expected URL to contain key parameter")
	}

	if !strings.Contains(pairData.URL, "path=") {
		t.Error("Expected URL to contain path parameter")
	}

	if !strings.Contains(pairData.URL, "expires=") {
		t.Error("Expected URL to contain expires parameter")
	}
}

func TestIsValidPairURL_ValidURL(t *testing.T) {
	validURLs := []string{
		"fybrk://pair?key=abc123",
		"fybrk://pair?key=abc&path=/test",
		"fybrk://pair?key=abc&path=/test&expires=123456",
	}

	for _, url := range validURLs {
		if !IsValidPairURL(url) {
			t.Errorf("Expected %s to be valid", url)
		}
	}
}

func TestIsValidPairURL_InvalidURL(t *testing.T) {
	invalidURLs := []string{
		"",
		"fybrk://",
		"fybrk://pair",
		"http://example.com",
		"fybrk://other?key=abc",
		"not-a-url",
	}

	for _, url := range invalidURLs {
		if IsValidPairURL(url) {
			t.Errorf("Expected %s to be invalid", url)
		}
	}
}

func TestJoinFromPairData_ValidURL(t *testing.T) {
	tempDir := t.TempDir()
	pairURL := "fybrk://pair?key=abc123&path=/remote&expires=123456"

	fybrk, err := JoinFromPairData(pairURL, tempDir)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	if fybrk.GetSyncPath() != tempDir {
		t.Errorf("Expected sync path %s, got %s", tempDir, fybrk.GetSyncPath())
	}
}

func TestJoinFromPairData_InvalidURL(t *testing.T) {
	tempDir := t.TempDir()
	invalidURL := "not-a-fybrk-url"

	_, err := JoinFromPairData(invalidURL, tempDir)

	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "invalid pairing URL format") {
		t.Errorf("Expected 'invalid pairing URL format' error, got: %v", err)
	}
}

func TestJoinFromPairData_DefaultPath(t *testing.T) {
	pairURL := "fybrk://pair?key=abc123&path=/remote&expires=123456"

	fybrk, err := JoinFromPairData(pairURL, "")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer fybrk.Close()

	// Should use current directory when no path specified
	currentDir, _ := os.Getwd()
	if fybrk.GetSyncPath() != currentDir {
		t.Errorf("Expected current directory %s, got %s", currentDir, fybrk.GetSyncPath())
	}
}

func TestClose_DatabaseConnection(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify database is open
	if err := fybrk.db.Ping(); err != nil {
		t.Errorf("Database should be open: %v", err)
	}

	// Close and verify
	err = fybrk.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}

	// Database should be closed (ping should fail)
	if err := fybrk.db.Ping(); err == nil {
		t.Error("Database should be closed after Close()")
	}
}

func TestMultipleInstances_SameDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create first instance
	config1 := Config{SyncPath: tempDir}
	fybrk1, err := New(config1)
	if err != nil {
		t.Fatalf("Expected no error for first instance, got: %v", err)
	}
	defer fybrk1.Close()

	// Create second instance (should work with WAL mode)
	config2 := Config{SyncPath: tempDir}
	fybrk2, err := New(config2)
	if err != nil {
		t.Fatalf("Expected no error for second instance, got: %v", err)
	}
	defer fybrk2.Close()

	// Both should have the same key
	key1 := fybrk1.GetKey()
	key2 := fybrk2.GetKey()

	if len(key1) != len(key2) {
		t.Error("Keys should have same length")
	}

	for i := range key1 {
		if key1[i] != key2[i] {
			t.Error("Keys should be identical")
			break
		}
	}
}
