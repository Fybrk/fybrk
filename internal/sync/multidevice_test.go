package sync

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Fybrk/fybrk/internal/network"
	"github.com/Fybrk/fybrk/internal/storage"
	"github.com/Fybrk/fybrk/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMultiDeviceTest(t *testing.T) (*MultiDeviceSync, func()) {
	tmpDir := t.TempDir()
	syncPath := filepath.Join(tmpDir, "sync")
	dbPath := filepath.Join(tmpDir, "metadata.db")

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

	engine, err := NewEngine(metadataStore, chunker, encryptor, syncPath, "test-device")
	require.NoError(t, err)

	mds, err := NewMultiDeviceSync(engine, encryptor, "test-device", 0)
	require.NoError(t, err)

	cleanup := func() {
		mds.Stop()
		engine.Close()
		metadataStore.Close()
	}

	return mds, cleanup
}

func TestNewMultiDeviceSync(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	assert.NotNil(t, mds)
	assert.NotNil(t, mds.engine)
	assert.NotNil(t, mds.network)
	assert.NotNil(t, mds.encryptor)
	assert.Equal(t, "test-device", mds.deviceID)
}

func TestMultiDeviceSyncStartStop(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	err := mds.Start()
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	err = mds.Stop()
	assert.NoError(t, err)
}

func TestGetConnectedDevices(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	devices := mds.GetConnectedDevices()
	assert.Empty(t, devices)
}

func TestHandleMessage(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	// Test that message handler is set
	assert.NotNil(t, mds.network)

	// Test non-sync message (should be ignored)
	msg := &network.Message{
		Type:     "other",
		DeviceID: "sender",
		Data:     "test",
	}

	mds.handleMessage("sender", msg)
	// Should not panic or error
}

func TestBroadcastFileList(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	// Create a test file
	testFile := filepath.Join(mds.engine.syncPath, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Scan to add file to engine
	err = mds.engine.ScanDirectory()
	require.NoError(t, err)

	// Test broadcast (won't actually send since no peers connected)
	mds.broadcastFileList()

	// Verify files exist in engine
	files, err := mds.engine.GetSyncedFiles()
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestHandleFileList(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	// Create mock remote files
	remoteFiles := []*types.FileMetadata{
		{
			Path:    "remote-file.txt",
			Version: 1,
			Size:    100,
			Chunks:  [][32]byte{{1, 2, 3}}, // Mock chunk hash
		},
	}

	// Handle file list (should trigger file request)
	mds.handleFileList("remote-device", remoteFiles)

	// Verify no errors occurred
	// In a real implementation, this would trigger network requests
}

func TestRequestFile(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	fileMetadata := &types.FileMetadata{
		Path:   "test-file.txt",
		Chunks: [][32]byte{{1, 2, 3}},
	}

	// Should not panic when requesting from non-existent device
	mds.requestFile("non-existent-device", fileMetadata)
}

func TestSyncMessageTypes(t *testing.T) {
	// Test SyncMessage serialization
	syncMsg := SyncMessage{
		Type: "file_list",
		Files: []*types.FileMetadata{
			{
				Path:    "test.txt",
				Version: 1,
			},
		},
	}

	data, err := json.Marshal(syncMsg)
	require.NoError(t, err)

	var decoded SyncMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "file_list", decoded.Type)
	assert.Len(t, decoded.Files, 1)
	assert.Equal(t, "test.txt", decoded.Files[0].Path)
}

func TestFileRequestResponse(t *testing.T) {
	// Test FileRequest serialization
	request := FileRequest{
		Path:   "test.txt",
		Chunks: [][32]byte{{1, 2, 3}},
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	var decoded FileRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "test.txt", decoded.Path)
	assert.Len(t, decoded.Chunks, 1)
}

func TestHandleFileRequest(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	request := &FileRequest{
		Path:   "non-existent.txt",
		Chunks: [][32]byte{{1, 2, 3}},
	}

	// Should handle gracefully even if file doesn't exist
	mds.handleFileRequest("requester-device", request)
	// No assertion needed - just verify no panic
}

func TestHandleFileResponse(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	response := &FileResponse{
		Path:   "received-file.txt",
		Chunks: []types.Chunk{},
	}

	// Should handle gracefully even with empty chunks
	mds.handleFileResponse("sender-device", response)
	// No assertion needed - just verify no panic
}

func TestPeriodicSync(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	// Start the network to enable periodic sync
	err := mds.Start()
	require.NoError(t, err)

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	err = mds.Stop()
	assert.NoError(t, err)
}

func TestMultiDeviceIntegration(t *testing.T) {
	// Create two multi-device sync instances
	mds1, cleanup1 := setupMultiDeviceTest(t)
	defer cleanup1()

	mds2, cleanup2 := setupMultiDeviceTest(t)
	defer cleanup2()

	// Start both
	err := mds1.Start()
	require.NoError(t, err)

	err = mds2.Start()
	require.NoError(t, err)

	// Let them run briefly
	time.Sleep(200 * time.Millisecond)

	// Stop both
	mds1.Stop()
	mds2.Stop()

	// Verify they started and stopped without errors
}

func TestGetFileChunks(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	// Test getting file chunks for non-existent file
	chunks, err := mds.getFileChunks("non-existent.txt", nil)
	assert.Error(t, err)
	assert.Nil(t, chunks)
	assert.Contains(t, err.Error(), "file not found")
}

func TestWriteReceivedFile(t *testing.T) {
	mds, cleanup := setupMultiDeviceTest(t)
	defer cleanup()

	// Test data
	testData := []byte("This is test data for file writing.")
	testPath := "received-test.txt"

	// Write received file
	err := mds.writeReceivedFile(testPath, testData)
	require.NoError(t, err)

	// Verify file was written to the sync directory
	expectedPath := filepath.Join(mds.engine.syncPath, "received-test.txt")
	writtenData, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Equal(t, testData, writtenData)
}
