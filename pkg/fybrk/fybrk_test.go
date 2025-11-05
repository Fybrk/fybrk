package fybrk

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tmpDir := t.TempDir()
	syncPath := filepath.Join(tmpDir, "sync")
	dbPath := filepath.Join(tmpDir, "test.db")
	
	err := os.MkdirAll(syncPath, 0755)
	require.NoError(t, err)
	
	key := make([]byte, 32)
	rand.Read(key)
	
	config := &Config{
		SyncPath:  syncPath,
		DBPath:    dbPath,
		DeviceID:  "test-device",
		ChunkSize: 1024,
		Key:       key,
	}
	
	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	
	defer client.Close()
}

func TestNewClientInvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := &Config{
		SyncPath:  tmpDir,
		DBPath:    filepath.Join(tmpDir, "test.db"),
		DeviceID:  "test-device",
		ChunkSize: 1024,
		Key:       []byte("invalid"), // Too short
	}
	
	_, err := NewClient(config)
	assert.Error(t, err)
}

func TestClientScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	syncPath := filepath.Join(tmpDir, "sync")
	dbPath := filepath.Join(tmpDir, "test.db")
	
	err := os.MkdirAll(syncPath, 0755)
	require.NoError(t, err)
	
	// Create test file
	testFile := filepath.Join(syncPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	key := make([]byte, 32)
	rand.Read(key)
	
	config := &Config{
		SyncPath:  syncPath,
		DBPath:    dbPath,
		DeviceID:  "test-device",
		ChunkSize: 1024,
		Key:       key,
	}
	
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()
	
	err = client.ScanDirectory()
	require.NoError(t, err)
	
	files, err := client.GetSyncedFiles()
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "test.txt", files[0].Path)
}

func TestClientGetSyncedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	syncPath := filepath.Join(tmpDir, "sync")
	dbPath := filepath.Join(tmpDir, "test.db")
	
	err := os.MkdirAll(syncPath, 0755)
	require.NoError(t, err)
	
	key := make([]byte, 32)
	rand.Read(key)
	
	config := &Config{
		SyncPath:  syncPath,
		DBPath:    dbPath,
		DeviceID:  "test-device",
		ChunkSize: 1024,
		Key:       key,
	}
	
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()
	
	// Initially no files
	files, err := client.GetSyncedFiles()
	require.NoError(t, err)
	assert.Len(t, files, 0)
}

func TestClientMultiDeviceSync(t *testing.T) {
	tmpDir := t.TempDir()
	syncPath := filepath.Join(tmpDir, "sync")
	dbPath := filepath.Join(tmpDir, "test.db")
	
	err := os.MkdirAll(syncPath, 0755)
	require.NoError(t, err)
	
	key := make([]byte, 32)
	rand.Read(key)
	
	config := &Config{
		SyncPath:  syncPath,
		DBPath:    dbPath,
		DeviceID:  "test-device",
		ChunkSize: 1024,
		Key:       key,
	}
	
	client, err := NewClient(config)
	require.NoError(t, err)
	defer client.Close()
	
	// Enable multi-device sync
	err = client.EnableMultiDeviceSync(0) // Use random port
	require.NoError(t, err)
	
	// Initially no connected devices
	devices := client.GetConnectedDevices()
	assert.Empty(t, devices)
}

func TestClientClose(t *testing.T) {
	tmpDir := t.TempDir()
	syncPath := filepath.Join(tmpDir, "sync")
	dbPath := filepath.Join(tmpDir, "test.db")
	
	err := os.MkdirAll(syncPath, 0755)
	require.NoError(t, err)
	
	key := make([]byte, 32)
	rand.Read(key)
	
	config := &Config{
		SyncPath:  syncPath,
		DBPath:    dbPath,
		DeviceID:  "test-device",
		ChunkSize: 1024,
		Key:       key,
	}
	
	client, err := NewClient(config)
	require.NoError(t, err)
	
	err = client.Close()
	assert.NoError(t, err)
}
