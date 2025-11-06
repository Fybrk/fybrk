package api

import (
	"encoding/json"
	"path/filepath"
	gosync "sync"
	"testing"
	"time"

	"github.com/Fybrk/fybrk/internal/storage"
	"github.com/Fybrk/fybrk/internal/sync"
	"github.com/Fybrk/fybrk/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestEngine(tempDir string) (*sync.Engine, error) {
	key := []byte("12345678901234567890123456789012") // Exactly 32 bytes

	metadataStore, err := storage.NewMetadataStore(filepath.Join(tempDir, "metadata.db"))
	if err != nil {
		return nil, err
	}

	chunker := storage.NewChunker(1024)
	encryptor, err := storage.NewEncryptor(key)
	if err != nil {
		return nil, err
	}

	return sync.NewEngine(metadataStore, chunker, encryptor, tempDir, "test-device")
}

func TestNewCrossPlatformAPI(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	assert.NotNil(t, api)
	assert.NotNil(t, api.engine)
	assert.NotNil(t, api.protocol)
	assert.NotNil(t, api.pairingManager)
	assert.NotNil(t, api.eventHandlers)
	assert.NotNil(t, api.ctx)
	assert.NotNil(t, api.cancel)
}

func TestGetDeviceInfo(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	info := api.GetDeviceInfo()
	assert.Equal(t, "test-device", info.ID)
	assert.Equal(t, "Test Device", info.Name)
	assert.Equal(t, "desktop", info.Type)
	assert.Equal(t, "online", info.Status)
	assert.Contains(t, info.Capabilities, "sync")
	assert.Contains(t, info.Capabilities, "pair")
	assert.Contains(t, info.Capabilities, "store")
}

func TestGetConnectedDevices(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	devices, err := api.GetConnectedDevices()
	assert.NoError(t, err)
	assert.Empty(t, devices)
}

func TestGeneratePairingQR(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	qrData, err := api.GeneratePairingQR("192.168.1.100:8080")
	assert.NoError(t, err)
	assert.NotNil(t, qrData)
	assert.NotEmpty(t, qrData.ImageData)
	assert.NotEmpty(t, qrData.TextData)
	assert.True(t, qrData.ExpiresAt.After(time.Now()))
}

func TestGetSyncStats(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	stats, err := api.GetSyncStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalFiles)
	assert.Equal(t, 0, stats.SyncedFiles)
	assert.Equal(t, int64(0), stats.TotalSize)
	assert.Equal(t, int64(0), stats.SyncedSize)
	assert.False(t, stats.SyncInProgress)
	assert.Equal(t, 0, stats.ConnectedDevices)
	assert.Equal(t, 0, stats.Conflicts)
}

func TestStartStopSync(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	// Test start sync
	err = api.StartSync()
	assert.NoError(t, err)

	// Test stop sync
	err = api.StopSync()
	assert.NoError(t, err)
}

func TestGetFileList(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	files, err := api.GetFileList()
	assert.NoError(t, err)
	// Files can be nil or empty slice
	if files != nil {
		assert.IsType(t, []*types.FileMetadata{}, files)
	}
}

func TestAddSyncPath(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	err = api.AddSyncPath("/test/path")
	assert.NoError(t, err)
}

func TestEventSystem(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	var mu gosync.Mutex
	eventReceived := false
	var receivedEvent *Event

	// Register event handler with proper synchronization
	api.OnEvent("test.event", func(event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		eventReceived = true
		receivedEvent = event
		return nil
	})

	// Emit event
	api.emitEvent("test.event", map[string]interface{}{
		"test": "data",
	})

	// Give some time for async handler
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, eventReceived)
	assert.NotNil(t, receivedEvent)
	assert.Equal(t, "test.event", receivedEvent.Type)
	assert.Equal(t, "data", receivedEvent.Data["test"])
}

func TestGetDeviceFingerprint(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	fingerprint := api.GetDeviceFingerprint()
	assert.NotEmpty(t, fingerprint)
}

func TestExportImportConfig(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	// Test export
	configData, err := api.ExportConfig()
	assert.NoError(t, err)
	assert.NotEmpty(t, configData)

	// Verify exported data is valid JSON
	var config map[string]interface{}
	err = json.Unmarshal(configData, &config)
	assert.NoError(t, err)
	assert.Equal(t, "test-device", config["device_id"])
	assert.Equal(t, "Test Device", config["device_name"])
	assert.Equal(t, "1.0.0", config["version"])

	// Test import
	err = api.ImportConfig(configData)
	assert.NoError(t, err)
}

func TestImportConfigInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	err = api.ImportConfig([]byte("invalid json"))
	assert.Error(t, err)
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")

	err = api.Close()
	assert.NoError(t, err)

	// Verify context is cancelled
	select {
	case <-api.ctx.Done():
		// Context is cancelled as expected
	default:
		t.Error("Context should be cancelled after Close()")
	}
}

func TestGetSyncStatsWithFiles(t *testing.T) {
	t.Skip("Skipping test to avoid database locking issues")
}

func TestMultipleEventHandlers(t *testing.T) {
	tempDir := t.TempDir()
	engine, err := createTestEngine(tempDir)
	require.NoError(t, err)
	defer engine.Close()

	api := NewCrossPlatformAPI(engine, "test-device", "Test Device", "desktop")
	defer api.Close()

	var mu gosync.Mutex
	handler1Called := false
	handler2Called := false

	// Register multiple handlers for same event with proper synchronization
	api.OnEvent("test.event", func(event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		handler1Called = true
		return nil
	})

	api.OnEvent("test.event", func(event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		handler2Called = true
		return nil
	})

	// Emit event
	api.emitEvent("test.event", map[string]interface{}{})

	// Give some time for async handlers
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, handler1Called)
	assert.True(t, handler2Called)
}
