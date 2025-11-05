package protocol

import (
	"encoding/json"
	"time"

	"github.com/Fybrk/fybrk/pkg/types"
)

// UniversalSyncProtocol defines the cross-platform sync protocol
type UniversalSyncProtocol struct {
	version    int
	deviceID   string
	deviceType string
}

// SyncMessage represents any sync message in the universal protocol
type SyncMessage struct {
	Version   int                    `json:"version"`
	Type      string                 `json:"type"`
	DeviceID  string                 `json:"device_id"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// Message types for the universal protocol
const (
	// Device management
	MsgDeviceAnnounce   = "device.announce"
	MsgDevicePair       = "device.pair"
	MsgDeviceHeartbeat  = "device.heartbeat"
	MsgDeviceCapability = "device.capability"

	// File operations
	MsgFileList     = "file.list"
	MsgFileRequest  = "file.request"
	MsgFileResponse = "file.response"
	MsgFileUpdate   = "file.update"
	MsgFileDelete   = "file.delete"

	// Sync operations
	MsgSyncStart    = "sync.start"
	MsgSyncComplete = "sync.complete"
	MsgSyncConflict = "sync.conflict"
	MsgSyncStatus   = "sync.status"

	// Future: AI and advanced features
	MsgAIQuery      = "ai.query"
	MsgAIResponse   = "ai.response"
	MsgSearchQuery  = "search.query"
	MsgSearchResult = "search.result"
)

// DeviceAnnouncement for device discovery
type DeviceAnnouncement struct {
	DeviceID     string            `json:"device_id"`
	DeviceName   string            `json:"device_name"`
	DeviceType   string            `json:"device_type"`
	Capabilities []string          `json:"capabilities"`
	NetworkInfo  map[string]string `json:"network_info"`
	PublicKey    string            `json:"public_key"`
}

// FileOperation represents file sync operations
type FileOperation struct {
	Operation string                `json:"operation"` // create, update, delete
	Path      string                `json:"path"`
	Metadata  *types.FileMetadata   `json:"metadata,omitempty"`
	Chunks    []types.Chunk         `json:"chunks,omitempty"`
	Conflict  *ConflictInfo         `json:"conflict,omitempty"`
}

// ConflictInfo represents sync conflicts
type ConflictInfo struct {
	ConflictType string            `json:"conflict_type"` // timestamp, content, delete
	LocalFile    *types.FileMetadata `json:"local_file"`
	RemoteFile   *types.FileMetadata `json:"remote_file"`
	Resolution   string            `json:"resolution"` // local_wins, remote_wins, merge, manual
}

// SyncStatus represents sync state
type SyncStatus struct {
	State        string    `json:"state"` // idle, syncing, conflict, error
	Progress     float64   `json:"progress"`
	FilesTotal   int       `json:"files_total"`
	FilesSynced  int       `json:"files_synced"`
	BytesTotal   int64     `json:"bytes_total"`
	BytesSynced  int64     `json:"bytes_synced"`
	LastSync     time.Time `json:"last_sync"`
	Conflicts    int       `json:"conflicts"`
	Errors       []string  `json:"errors,omitempty"`
}

func NewUniversalSyncProtocol(deviceID, deviceType string) *UniversalSyncProtocol {
	return &UniversalSyncProtocol{
		version:    1,
		deviceID:   deviceID,
		deviceType: deviceType,
	}
}

// CreateMessage creates a new sync message
func (usp *UniversalSyncProtocol) CreateMessage(msgType string, payload interface{}) (*SyncMessage, error) {
	payloadMap := make(map[string]interface{})
	
	// Convert payload to map
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payloadBytes, &payloadMap); err != nil {
			return nil, err
		}
	}

	return &SyncMessage{
		Version:   usp.version,
		Type:      msgType,
		DeviceID:  usp.deviceID,
		Timestamp: time.Now(),
		Payload:   payloadMap,
	}, nil
}

// ParseMessage parses a sync message and extracts payload
func (usp *UniversalSyncProtocol) ParseMessage(msg *SyncMessage, target interface{}) error {
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(payloadBytes, target)
}

// CreateDeviceAnnouncement creates a device announcement message
func (usp *UniversalSyncProtocol) CreateDeviceAnnouncement(name string, capabilities []string, publicKey string) (*SyncMessage, error) {
	announcement := DeviceAnnouncement{
		DeviceID:     usp.deviceID,
		DeviceName:   name,
		DeviceType:   usp.deviceType,
		Capabilities: capabilities,
		NetworkInfo: map[string]string{
			"protocol": "fybrk-v1",
			"platform": usp.deviceType,
		},
		PublicKey: publicKey,
	}

	return usp.CreateMessage(MsgDeviceAnnounce, announcement)
}

// CreateFileOperation creates a file operation message
func (usp *UniversalSyncProtocol) CreateFileOperation(operation, path string, metadata *types.FileMetadata, chunks []types.Chunk) (*SyncMessage, error) {
	fileOp := FileOperation{
		Operation: operation,
		Path:      path,
		Metadata:  metadata,
		Chunks:    chunks,
	}

	return usp.CreateMessage(MsgFileUpdate, fileOp)
}

// CreateSyncStatus creates a sync status message
func (usp *UniversalSyncProtocol) CreateSyncStatus(status SyncStatus) (*SyncMessage, error) {
	return usp.CreateMessage(MsgSyncStatus, status)
}

// IsCompatible checks if a message is compatible with this protocol version
func (usp *UniversalSyncProtocol) IsCompatible(msg *SyncMessage) bool {
	return msg.Version <= usp.version
}

// GetSupportedMessageTypes returns all supported message types
func (usp *UniversalSyncProtocol) GetSupportedMessageTypes() []string {
	return []string{
		MsgDeviceAnnounce, MsgDevicePair, MsgDeviceHeartbeat, MsgDeviceCapability,
		MsgFileList, MsgFileRequest, MsgFileResponse, MsgFileUpdate, MsgFileDelete,
		MsgSyncStart, MsgSyncComplete, MsgSyncConflict, MsgSyncStatus,
		MsgAIQuery, MsgAIResponse, MsgSearchQuery, MsgSearchResult,
	}
}

// GetDeviceID returns the device ID
func (usp *UniversalSyncProtocol) GetDeviceID() string {
	return usp.deviceID
}

// GetDeviceType returns the device type
func (usp *UniversalSyncProtocol) GetDeviceType() string {
	return usp.deviceType
}

// Future: AI Integration structures
type AIQuery struct {
	Query     string            `json:"query"`
	Context   map[string]string `json:"context"`
	ModelType string            `json:"model_type"` // local, hybrid, none
}

type AIResponse struct {
	Response  string            `json:"response"`
	Sources   []string          `json:"sources"`
	Confidence float64          `json:"confidence"`
}

// Future: Search structures
type SearchQuery struct {
	Query     string   `json:"query"`
	FileTypes []string `json:"file_types"`
	DateRange *struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"date_range,omitempty"`
}

type SearchResult struct {
	Files   []*types.FileMetadata `json:"files"`
	Matches []struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Context string `json:"context"`
	} `json:"matches"`
}
