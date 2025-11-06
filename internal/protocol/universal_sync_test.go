package protocol

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/Fybrk/fybrk/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUniversalSyncProtocol(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	assert.NotNil(t, usp)
	assert.Equal(t, 1, usp.version)
	assert.Equal(t, "test-device", usp.deviceID)
	assert.Equal(t, "desktop", usp.deviceType)
}

func TestCreateMessage(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	payload := map[string]interface{}{
		"test":   "data",
		"number": 42,
	}

	msg, err := usp.CreateMessage(MsgDeviceAnnounce, payload)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, 1, msg.Version)
	assert.Equal(t, MsgDeviceAnnounce, msg.Type)
	assert.Equal(t, "test-device", msg.DeviceID)
	assert.Equal(t, "data", msg.Payload["test"])
	assert.Equal(t, float64(42), msg.Payload["number"]) // JSON unmarshaling converts to float64
	assert.True(t, msg.Timestamp.Before(time.Now().Add(time.Second)))
}

func TestCreateMessageNilPayload(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	msg, err := usp.CreateMessage(MsgDeviceHeartbeat, nil)
	require.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, MsgDeviceHeartbeat, msg.Type)
	assert.Empty(t, msg.Payload)
}

func TestParseMessage(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	// Create a message with structured payload
	announcement := DeviceAnnouncement{
		DeviceID:     "test-device",
		DeviceName:   "Test Device",
		DeviceType:   "desktop",
		Capabilities: []string{"sync", "pair"},
		NetworkInfo:  map[string]string{"protocol": "fybrk-v1"},
		PublicKey:    "test-key",
	}

	msg, err := usp.CreateMessage(MsgDeviceAnnounce, announcement)
	require.NoError(t, err)

	// Parse the message back
	var parsed DeviceAnnouncement
	err = usp.ParseMessage(msg, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test-device", parsed.DeviceID)
	assert.Equal(t, "Test Device", parsed.DeviceName)
	assert.Equal(t, "desktop", parsed.DeviceType)
	assert.Equal(t, []string{"sync", "pair"}, parsed.Capabilities)
	assert.Equal(t, "fybrk-v1", parsed.NetworkInfo["protocol"])
	assert.Equal(t, "test-key", parsed.PublicKey)
}

func TestCreateDeviceAnnouncement(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	capabilities := []string{"sync", "pair", "store"}
	publicKey := "test-public-key"

	msg, err := usp.CreateDeviceAnnouncement("Test Device", capabilities, publicKey)
	require.NoError(t, err)

	assert.Equal(t, MsgDeviceAnnounce, msg.Type)
	assert.Equal(t, "test-device", msg.DeviceID)

	var announcement DeviceAnnouncement
	err = usp.ParseMessage(msg, &announcement)
	require.NoError(t, err)

	assert.Equal(t, "test-device", announcement.DeviceID)
	assert.Equal(t, "Test Device", announcement.DeviceName)
	assert.Equal(t, "desktop", announcement.DeviceType)
	assert.Equal(t, capabilities, announcement.Capabilities)
	assert.Equal(t, "fybrk-v1", announcement.NetworkInfo["protocol"])
	assert.Equal(t, "desktop", announcement.NetworkInfo["platform"])
	assert.Equal(t, publicKey, announcement.PublicKey)
}

func TestCreateFileOperation(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	// Create proper hash
	hash := sha256.Sum256([]byte("test-hash"))
	chunkHash := sha256.Sum256([]byte("chunk-hash"))

	metadata := &types.FileMetadata{
		Path:    "test.txt",
		Size:    100,
		ModTime: time.Now(),
		Hash:    hash,
		Version: 1,
		Chunks:  [][32]byte{chunkHash},
	}

	chunks := []types.Chunk{
		{
			Hash:      chunkHash,
			Size:      100,
			Data:      []byte("test data"),
			Encrypted: false,
			CreatedAt: time.Now(),
		},
	}

	msg, err := usp.CreateFileOperation("create", "test.txt", metadata, chunks)
	require.NoError(t, err)

	assert.Equal(t, MsgFileUpdate, msg.Type)

	var fileOp FileOperation
	err = usp.ParseMessage(msg, &fileOp)
	require.NoError(t, err)

	assert.Equal(t, "create", fileOp.Operation)
	assert.Equal(t, "test.txt", fileOp.Path)
	assert.NotNil(t, fileOp.Metadata)
	assert.Equal(t, "test.txt", fileOp.Metadata.Path)
	assert.Equal(t, int64(100), fileOp.Metadata.Size)
	assert.Len(t, fileOp.Chunks, 1)
	assert.Equal(t, chunkHash, fileOp.Chunks[0].Hash)
}

func TestCreateSyncStatus(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	status := SyncStatus{
		State:       "syncing",
		Progress:    0.75,
		FilesTotal:  100,
		FilesSynced: 75,
		BytesTotal:  1000000,
		BytesSynced: 750000,
		LastSync:    time.Now(),
		Conflicts:   2,
		Errors:      []string{"error1", "error2"},
	}

	msg, err := usp.CreateSyncStatus(status)
	require.NoError(t, err)

	assert.Equal(t, MsgSyncStatus, msg.Type)

	var parsed SyncStatus
	err = usp.ParseMessage(msg, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "syncing", parsed.State)
	assert.Equal(t, 0.75, parsed.Progress)
	assert.Equal(t, 100, parsed.FilesTotal)
	assert.Equal(t, 75, parsed.FilesSynced)
	assert.Equal(t, int64(1000000), parsed.BytesTotal)
	assert.Equal(t, int64(750000), parsed.BytesSynced)
	assert.Equal(t, 2, parsed.Conflicts)
	assert.Equal(t, []string{"error1", "error2"}, parsed.Errors)
}

func TestIsCompatible(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	// Compatible message (same version)
	msg1 := &SyncMessage{Version: 1}
	assert.True(t, usp.IsCompatible(msg1))

	// Compatible message (older version)
	msg2 := &SyncMessage{Version: 0}
	assert.True(t, usp.IsCompatible(msg2))

	// Incompatible message (newer version)
	msg3 := &SyncMessage{Version: 2}
	assert.False(t, usp.IsCompatible(msg3))
}

func TestGetSupportedMessageTypes(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	types := usp.GetSupportedMessageTypes()
	assert.NotEmpty(t, types)

	// Check for key message types
	assert.Contains(t, types, MsgDeviceAnnounce)
	assert.Contains(t, types, MsgFileUpdate)
	assert.Contains(t, types, MsgSyncStatus)
	assert.Contains(t, types, MsgAIQuery)
	assert.Contains(t, types, MsgSearchQuery)

	// Should have all defined message types
	expectedTypes := []string{
		MsgDeviceAnnounce, MsgDevicePair, MsgDeviceHeartbeat, MsgDeviceCapability,
		MsgFileList, MsgFileRequest, MsgFileResponse, MsgFileUpdate, MsgFileDelete,
		MsgSyncStart, MsgSyncComplete, MsgSyncConflict, MsgSyncStatus,
		MsgAIQuery, MsgAIResponse, MsgSearchQuery, MsgSearchResult,
	}

	for _, expectedType := range expectedTypes {
		assert.Contains(t, types, expectedType)
	}
}

func TestGetDeviceID(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device-123", "desktop")
	assert.Equal(t, "test-device-123", usp.GetDeviceID())
}

func TestGetDeviceType(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "mobile")
	assert.Equal(t, "mobile", usp.GetDeviceType())
}

func TestConflictInfo(t *testing.T) {
	localHash := sha256.Sum256([]byte("local-hash"))
	remoteHash := sha256.Sum256([]byte("remote-hash"))

	localFile := &types.FileMetadata{
		Path:    "test.txt",
		Size:    100,
		ModTime: time.Now(),
		Hash:    localHash,
		Version: 1,
	}

	remoteFile := &types.FileMetadata{
		Path:    "test.txt",
		Size:    200,
		ModTime: time.Now().Add(time.Hour),
		Hash:    remoteHash,
		Version: 2,
	}

	conflict := ConflictInfo{
		ConflictType: "content",
		LocalFile:    localFile,
		RemoteFile:   remoteFile,
		Resolution:   "manual",
	}

	assert.Equal(t, "content", conflict.ConflictType)
	assert.Equal(t, localHash, conflict.LocalFile.Hash)
	assert.Equal(t, remoteHash, conflict.RemoteFile.Hash)
	assert.Equal(t, "manual", conflict.Resolution)
}

func TestAIQueryResponse(t *testing.T) {
	query := AIQuery{
		Query:     "Find all documents about project X",
		Context:   map[string]string{"project": "X", "type": "documents"},
		ModelType: "local",
	}

	response := AIResponse{
		Response:   "Found 5 documents about project X",
		Sources:    []string{"doc1.txt", "doc2.pdf"},
		Confidence: 0.95,
	}

	assert.Equal(t, "Find all documents about project X", query.Query)
	assert.Equal(t, "X", query.Context["project"])
	assert.Equal(t, "local", query.ModelType)

	assert.Equal(t, "Found 5 documents about project X", response.Response)
	assert.Equal(t, []string{"doc1.txt", "doc2.pdf"}, response.Sources)
	assert.Equal(t, 0.95, response.Confidence)
}

func TestSearchQueryResult(t *testing.T) {
	searchQuery := SearchQuery{
		Query:     "test content",
		FileTypes: []string{".txt", ".md"},
		DateRange: &struct {
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		}{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
	}

	files := []*types.FileMetadata{
		{
			Path: "test1.txt",
			Size: 100,
		},
	}

	matches := []struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Context string `json:"context"`
	}{
		{
			File:    "test1.txt",
			Line:    5,
			Context: "This is test content in the file",
		},
	}

	result := SearchResult{
		Files:   files,
		Matches: matches,
	}

	assert.Equal(t, "test content", searchQuery.Query)
	assert.Equal(t, []string{".txt", ".md"}, searchQuery.FileTypes)
	assert.NotNil(t, searchQuery.DateRange)

	assert.Len(t, result.Files, 1)
	assert.Equal(t, "test1.txt", result.Files[0].Path)
	assert.Len(t, result.Matches, 1)
	assert.Equal(t, "test1.txt", result.Matches[0].File)
	assert.Equal(t, 5, result.Matches[0].Line)
}

func TestMessageConstants(t *testing.T) {
	// Test that all message type constants are defined
	assert.Equal(t, "device.announce", MsgDeviceAnnounce)
	assert.Equal(t, "device.pair", MsgDevicePair)
	assert.Equal(t, "device.heartbeat", MsgDeviceHeartbeat)
	assert.Equal(t, "device.capability", MsgDeviceCapability)

	assert.Equal(t, "file.list", MsgFileList)
	assert.Equal(t, "file.request", MsgFileRequest)
	assert.Equal(t, "file.response", MsgFileResponse)
	assert.Equal(t, "file.update", MsgFileUpdate)
	assert.Equal(t, "file.delete", MsgFileDelete)

	assert.Equal(t, "sync.start", MsgSyncStart)
	assert.Equal(t, "sync.complete", MsgSyncComplete)
	assert.Equal(t, "sync.conflict", MsgSyncConflict)
	assert.Equal(t, "sync.status", MsgSyncStatus)

	assert.Equal(t, "ai.query", MsgAIQuery)
	assert.Equal(t, "ai.response", MsgAIResponse)
	assert.Equal(t, "search.query", MsgSearchQuery)
	assert.Equal(t, "search.result", MsgSearchResult)
}

func TestComplexPayloadSerialization(t *testing.T) {
	usp := NewUniversalSyncProtocol("test-device", "desktop")

	// Test with complex nested structure
	complexPayload := map[string]interface{}{
		"string":  "test",
		"number":  42,
		"boolean": true,
		"array":   []interface{}{1, 2, 3},
		"nested": map[string]interface{}{
			"inner": "value",
		},
	}

	msg, err := usp.CreateMessage("test.complex", complexPayload)
	require.NoError(t, err)

	// Verify the payload was serialized correctly
	assert.Equal(t, "test", msg.Payload["string"])
	assert.Equal(t, float64(42), msg.Payload["number"])
	assert.Equal(t, true, msg.Payload["boolean"])

	array, ok := msg.Payload["array"].([]interface{})
	require.True(t, ok)
	assert.Len(t, array, 3)

	nested, ok := msg.Payload["nested"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", nested["inner"])
}
