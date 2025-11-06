package storage

import (
	"crypto/sha256"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fybrk/fybrk/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetadataStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	require.NotNil(t, store)

	defer store.Close()

	// Verify tables were created
	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('files', 'devices')").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestStoreAndGetFileMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Create test metadata
	hash := sha256.Sum256([]byte("test content"))
	chunks := [][32]byte{hash}

	metadata := &types.FileMetadata{
		Path:    "/test/file.txt",
		Hash:    hash,
		Size:    100,
		ModTime: time.Now().UTC().Truncate(time.Second), // Use UTC and SQLite precision
		Chunks:  chunks,
		Version: 1,
	}

	// Store metadata
	err = store.StoreFileMetadata(metadata)
	require.NoError(t, err)

	// Retrieve metadata
	retrieved, err := store.GetFileMetadata("/test/file.txt")
	require.NoError(t, err)

	assert.Equal(t, metadata.Path, retrieved.Path)
	assert.Equal(t, metadata.Hash, retrieved.Hash)
	assert.Equal(t, metadata.Size, retrieved.Size)
	assert.Equal(t, metadata.ModTime, retrieved.ModTime)
	assert.Equal(t, metadata.Chunks, retrieved.Chunks)
	assert.Equal(t, metadata.Version, retrieved.Version)
}

func TestGetNonexistentFileMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.GetFileMetadata("/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Store multiple files
	files := []*types.FileMetadata{
		{
			Path:    "/test/file1.txt",
			Hash:    sha256.Sum256([]byte("content1")),
			Size:    10,
			ModTime: time.Now().UTC().Truncate(time.Second),
			Chunks:  [][32]byte{sha256.Sum256([]byte("content1"))},
			Version: 1,
		},
		{
			Path:    "/test/file2.txt",
			Hash:    sha256.Sum256([]byte("content2")),
			Size:    20,
			ModTime: time.Now().UTC().Truncate(time.Second),
			Chunks:  [][32]byte{sha256.Sum256([]byte("content2"))},
			Version: 1,
		},
	}

	for _, file := range files {
		err := store.StoreFileMetadata(file)
		require.NoError(t, err)
	}

	// List files
	retrieved, err := store.ListFiles()
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)

	// Files should be sorted by path
	assert.Equal(t, "/test/file1.txt", retrieved[0].Path)
	assert.Equal(t, "/test/file2.txt", retrieved[1].Path)
}

func TestDeleteFileMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Store metadata
	metadata := &types.FileMetadata{
		Path:    "/test/file.txt",
		Hash:    sha256.Sum256([]byte("content")),
		Size:    10,
		ModTime: time.Now().UTC().Truncate(time.Second),
		Chunks:  [][32]byte{sha256.Sum256([]byte("content"))},
		Version: 1,
	}

	err = store.StoreFileMetadata(metadata)
	require.NoError(t, err)

	// Verify it exists
	_, err = store.GetFileMetadata("/test/file.txt")
	require.NoError(t, err)

	// Delete it
	err = store.DeleteFileMetadata("/test/file.txt")
	require.NoError(t, err)

	// Verify it's gone
	_, err = store.GetFileMetadata("/test/file.txt")
	assert.Error(t, err)
}

func TestStoreAndGetDevice(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Create test device
	device := &types.Device{
		ID:       "device-123",
		Name:     "Test Device",
		Profile:  types.FullReplica,
		LastSeen: time.Now().UTC().Truncate(time.Second),
	}

	// Store device
	err = store.StoreDevice(device)
	require.NoError(t, err)

	// Retrieve device
	retrieved, err := store.GetDevice("device-123")
	require.NoError(t, err)

	assert.Equal(t, device.ID, retrieved.ID)
	assert.Equal(t, device.Name, retrieved.Name)
	assert.Equal(t, device.Profile, retrieved.Profile)
	assert.Equal(t, device.LastSeen, retrieved.LastSeen)
}

func TestGetNonexistentDevice(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	_, err = store.GetDevice("nonexistent-device")
	assert.Error(t, err)
}

func TestUpdateFileMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMetadataStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Store initial metadata
	metadata := &types.FileMetadata{
		Path:    "/test/file.txt",
		Hash:    sha256.Sum256([]byte("content")),
		Size:    10,
		ModTime: time.Now().UTC().Truncate(time.Second),
		Chunks:  [][32]byte{sha256.Sum256([]byte("content"))},
		Version: 1,
	}

	err = store.StoreFileMetadata(metadata)
	require.NoError(t, err)

	// Update metadata
	newHash := sha256.Sum256([]byte("new content"))
	metadata.Hash = newHash
	metadata.Size = 20
	metadata.Version = 2
	metadata.Chunks = [][32]byte{newHash}

	err = store.StoreFileMetadata(metadata)
	require.NoError(t, err)

	// Retrieve updated metadata
	retrieved, err := store.GetFileMetadata("/test/file.txt")
	require.NoError(t, err)

	assert.Equal(t, newHash, retrieved.Hash)
	assert.Equal(t, int64(20), retrieved.Size)
	assert.Equal(t, int64(2), retrieved.Version)
}
