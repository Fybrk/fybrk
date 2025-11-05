package sync

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fybrk/fybrk/internal/storage"
	"github.com/Fybrk/fybrk/internal/watcher"
	"github.com/Fybrk/fybrk/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEngine(t *testing.T) (*Engine, string, func()) {
	tmpDir := t.TempDir()
	syncPath := filepath.Join(tmpDir, "sync")
	dbPath := filepath.Join(tmpDir, fmt.Sprintf("metadata_%d.db", time.Now().UnixNano()))
	
	err := os.MkdirAll(syncPath, 0755)
	require.NoError(t, err)
	
	// Create components
	metadataStore, err := storage.NewMetadataStore(dbPath)
	require.NoError(t, err)
	
	chunker := storage.NewChunker(1024)
	
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, err := storage.NewEncryptor(key)
	require.NoError(t, err)
	
	engine, err := NewEngine(metadataStore, chunker, encryptor, syncPath, fmt.Sprintf("test-device-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	
	cleanup := func() {
		engine.Close()
		metadataStore.Close()
	}
	
	return engine, syncPath, cleanup
}

func TestNewEngine(t *testing.T) {
	engine, _, cleanup := setupTestEngine(t)
	defer cleanup()
	
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.metadataStore)
	assert.NotNil(t, engine.chunker)
	assert.NotNil(t, engine.encryptor)
	assert.NotNil(t, engine.watcher)
	assert.Contains(t, engine.deviceID, "test-device")
}

func TestScanDirectory(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "content of file 1",
		"file2.txt":        "content of file 2",
		"subdir/file3.txt": "content of file 3",
	}
	
	for relPath, content := range testFiles {
		fullPath := filepath.Join(syncPath, relPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}
	
	// Wait a bit for file system events to settle
	time.Sleep(200 * time.Millisecond)
	
	// Scan directory
	err := engine.ScanDirectory()
	require.NoError(t, err)
	
	// Verify files were processed
	files, err := engine.GetSyncedFiles()
	require.NoError(t, err)
	
	// Should have at least the files we created (may have more due to file watcher)
	assert.GreaterOrEqual(t, len(files), 3)
	
	// Check that our test files are present
	fileMap := make(map[string]*types.FileMetadata)
	for _, file := range files {
		fileMap[file.Path] = file
	}
	
	for relPath := range testFiles {
		normalizedPath := filepath.ToSlash(relPath)
		assert.Contains(t, fileMap, normalizedPath, "File %s should be present", normalizedPath)
		if metadata, exists := fileMap[normalizedPath]; exists {
			assert.Greater(t, len(metadata.Chunks), 0)
			assert.GreaterOrEqual(t, metadata.Version, int64(1))
		}
	}
}

func TestProcessFile(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	// Create test file
	testFile := filepath.Join(syncPath, "test.txt")
	testContent := "This is test content for processing"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	// Process the file
	err = engine.processFile(testFile, "test.txt")
	require.NoError(t, err)
	
	// Verify metadata was stored
	metadata, err := engine.metadataStore.GetFileMetadata("test.txt")
	require.NoError(t, err)
	
	assert.Equal(t, "test.txt", metadata.Path)
	assert.Equal(t, int64(len(testContent)), metadata.Size)
	assert.Greater(t, len(metadata.Chunks), 0)
	assert.Equal(t, int64(1), metadata.Version)
}

func TestFileVersioning(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	testFile := filepath.Join(syncPath, "versioned.txt")
	
	// Create initial file
	err := os.WriteFile(testFile, []byte("version 1"), 0644)
	require.NoError(t, err)
	
	// Wait for file watcher to process
	time.Sleep(200 * time.Millisecond)
	
	err = engine.processFile(testFile, "versioned.txt")
	require.NoError(t, err)
	
	// Check initial version
	metadata, err := engine.metadataStore.GetFileMetadata("versioned.txt")
	require.NoError(t, err)
	initialVersion := metadata.Version
	
	// Update file
	time.Sleep(100 * time.Millisecond) // Ensure different mod time
	err = os.WriteFile(testFile, []byte("version 2"), 0644)
	require.NoError(t, err)
	
	// Wait for file watcher to process
	time.Sleep(200 * time.Millisecond)
	
	err = engine.processFile(testFile, "versioned.txt")
	require.NoError(t, err)
	
	// Check updated version
	metadata, err = engine.metadataStore.GetFileMetadata("versioned.txt")
	require.NoError(t, err)
	assert.Greater(t, metadata.Version, initialVersion, "Version should have incremented")
}

func TestHandleFileChange(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	testFile := filepath.Join(syncPath, "change_test.txt")
	
	// Create file
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)
	
	// Handle file change
	engine.handleFileChange(testFile)
	
	// Give some time for processing
	time.Sleep(100 * time.Millisecond)
	
	// Verify file was processed
	metadata, err := engine.metadataStore.GetFileMetadata("change_test.txt")
	require.NoError(t, err)
	assert.Equal(t, "change_test.txt", metadata.Path)
}

func TestHandleFileRemoval(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	testFile := filepath.Join(syncPath, "remove_test.txt")
	
	// Create and process file
	err := os.WriteFile(testFile, []byte("to be removed"), 0644)
	require.NoError(t, err)
	
	err = engine.processFile(testFile, "remove_test.txt")
	require.NoError(t, err)
	
	// Verify file exists in metadata
	_, err = engine.metadataStore.GetFileMetadata("remove_test.txt")
	require.NoError(t, err)
	
	// Handle file removal
	engine.handleFileRemoval(testFile)
	
	// Verify file was removed from metadata
	_, err = engine.metadataStore.GetFileMetadata("remove_test.txt")
	assert.Error(t, err)
}

func TestFileEventProcessing(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	// Create file outside sync path (should be ignored)
	outsideFile := filepath.Join(filepath.Dir(syncPath), "outside.txt")
	err := os.WriteFile(outsideFile, []byte("outside"), 0644)
	require.NoError(t, err)
	
	// Process event for outside file
	engine.processFileEvent(watcher.FileEvent{
		Path:      outsideFile,
		Operation: "create",
	})
	
	// Should not be in metadata
	_, err = engine.metadataStore.GetFileMetadata("outside.txt")
	assert.Error(t, err)
	
	// Create file inside sync path
	insideFile := filepath.Join(syncPath, "inside.txt")
	err = os.WriteFile(insideFile, []byte("inside"), 0644)
	require.NoError(t, err)
	
	// Process event for inside file
	engine.processFileEvent(watcher.FileEvent{
		Path:      insideFile,
		Operation: "create",
	})
	
	// Give some time for processing
	time.Sleep(100 * time.Millisecond)
	
	// Should be in metadata
	_, err = engine.metadataStore.GetFileMetadata("inside.txt")
	require.NoError(t, err)
}

func TestDirectoryIgnoring(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	// Create directory
	testDir := filepath.Join(syncPath, "testdir")
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	
	// Handle directory change (should be ignored)
	engine.handleFileChange(testDir)
	
	// Directory should not be in metadata
	files, err := engine.GetSyncedFiles()
	require.NoError(t, err)
	assert.Len(t, files, 0)
}

func TestGetSyncedFiles(t *testing.T) {
	engine, syncPath, cleanup := setupTestEngine(t)
	defer cleanup()
	
	// Create multiple files
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(syncPath, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(testFile, []byte(fmt.Sprintf("content %d", i)), 0644)
		require.NoError(t, err)
		
		err = engine.processFile(testFile, fmt.Sprintf("file%d.txt", i))
		require.NoError(t, err)
	}
	
	// Get synced files
	files, err := engine.GetSyncedFiles()
	require.NoError(t, err)
	assert.Len(t, files, 3)
	
	// Files should be sorted by path
	for i, file := range files {
		expectedPath := fmt.Sprintf("file%d.txt", i)
		assert.Equal(t, expectedPath, file.Path)
	}
}
