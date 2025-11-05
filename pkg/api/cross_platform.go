package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Fybrk/fybrk/internal/pairing"
	"github.com/Fybrk/fybrk/internal/protocol"
	"github.com/Fybrk/fybrk/internal/sync"
	"github.com/Fybrk/fybrk/pkg/types"
)

// CrossPlatformAPI provides a unified API for desktop, mobile, and CLI apps
type CrossPlatformAPI struct {
	engine         *sync.Engine
	protocol       *protocol.UniversalSyncProtocol
	pairingManager *pairing.DevicePairingManager
	eventHandlers  map[string][]EventHandler
	ctx            context.Context
	cancel         context.CancelFunc
}

// EventHandler handles cross-platform events
type EventHandler func(event *Event) error

// Event represents a cross-platform event
type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// DeviceInfo represents device information for UI display
type DeviceInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Status       string    `json:"status"` // online, offline, syncing
	LastSeen     time.Time `json:"last_seen"`
	SyncProgress float64   `json:"sync_progress"`
	Capabilities []string  `json:"capabilities"`
}

// SyncStats represents sync statistics for UI display
type SyncStats struct {
	TotalFiles      int       `json:"total_files"`
	SyncedFiles     int       `json:"synced_files"`
	TotalSize       int64     `json:"total_size"`
	SyncedSize      int64     `json:"synced_size"`
	LastSync        time.Time `json:"last_sync"`
	SyncInProgress  bool      `json:"sync_in_progress"`
	ConnectedDevices int      `json:"connected_devices"`
	Conflicts       int       `json:"conflicts"`
}

// QRCodeData represents QR code data for device pairing
type QRCodeData struct {
	ImageData []byte `json:"image_data"`
	TextData  string `json:"text_data"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewCrossPlatformAPI(engine *sync.Engine, deviceID, deviceName, deviceType string) *CrossPlatformAPI {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &CrossPlatformAPI{
		engine:         engine,
		protocol:       protocol.NewUniversalSyncProtocol(deviceID, deviceType),
		pairingManager: pairing.NewDevicePairingManager(deviceID, deviceName),
		eventHandlers:  make(map[string][]EventHandler),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Device Management APIs

// GetDeviceInfo returns information about this device
func (api *CrossPlatformAPI) GetDeviceInfo() *DeviceInfo {
	return &DeviceInfo{
		ID:           api.protocol.GetDeviceID(),
		Name:         api.pairingManager.GetDeviceName(),
		Type:         api.protocol.GetDeviceType(),
		Status:       "online",
		LastSeen:     time.Now(),
		SyncProgress: 0.0,
		Capabilities: []string{"sync", "pair", "store"},
	}
}

// GetConnectedDevices returns list of connected devices
func (api *CrossPlatformAPI) GetConnectedDevices() ([]*DeviceInfo, error) {
	// This would get actual connected devices from the sync engine
	// For now, return empty list
	return []*DeviceInfo{}, nil
}

// Pairing APIs

// GeneratePairingQR generates a QR code for device pairing
func (api *CrossPlatformAPI) GeneratePairingQR(networkAddr string) (*QRCodeData, error) {
	qrImage, challenge, err := api.pairingManager.GeneratePairingQR(networkAddr)
	if err != nil {
		return nil, err
	}

	return &QRCodeData{
		ImageData: qrImage,
		TextData:  challenge,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}, nil
}

// PairWithQR pairs with another device using QR code data
func (api *CrossPlatformAPI) PairWithQR(qrData string) (*DeviceInfo, error) {
	pairingData, err := api.pairingManager.ParsePairingQR(qrData)
	if err != nil {
		return nil, err
	}

	device, err := api.pairingManager.InitiatePairing(pairingData)
	if err != nil {
		return nil, err
	}

	deviceInfo := &DeviceInfo{
		ID:           device.ID,
		Name:         device.Name,
		Type:         device.DeviceType,
		Status:       "paired",
		LastSeen:     device.LastSeen,
		SyncProgress: 0.0,
		Capabilities: device.Capabilities,
	}

	// Emit pairing event
	api.emitEvent("device.paired", map[string]interface{}{
		"device": deviceInfo,
	})

	return deviceInfo, nil
}

// Sync APIs

// GetSyncStats returns current sync statistics
func (api *CrossPlatformAPI) GetSyncStats() (*SyncStats, error) {
	files, err := api.engine.GetSyncedFiles()
	if err != nil {
		return nil, err
	}

	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}

	return &SyncStats{
		TotalFiles:       len(files),
		SyncedFiles:      len(files),
		TotalSize:        totalSize,
		SyncedSize:       totalSize,
		LastSync:         time.Now(),
		SyncInProgress:   false,
		ConnectedDevices: 0, // Would get from network layer
		Conflicts:        0,
	}, nil
}

// StartSync starts synchronization with all connected devices
func (api *CrossPlatformAPI) StartSync() error {
	// This would start the actual sync process
	api.emitEvent("sync.started", map[string]interface{}{
		"timestamp": time.Now(),
	})
	return nil
}

// StopSync stops synchronization
func (api *CrossPlatformAPI) StopSync() error {
	api.emitEvent("sync.stopped", map[string]interface{}{
		"timestamp": time.Now(),
	})
	return nil
}

// File Management APIs

// GetFileList returns list of synced files
func (api *CrossPlatformAPI) GetFileList() ([]*types.FileMetadata, error) {
	return api.engine.GetSyncedFiles()
}

// AddSyncPath adds a new path to sync
func (api *CrossPlatformAPI) AddSyncPath(path string) error {
	// This would add the path to the sync engine
	api.emitEvent("path.added", map[string]interface{}{
		"path": path,
	})
	return nil
}

// Event System APIs

// OnEvent registers an event handler
func (api *CrossPlatformAPI) OnEvent(eventType string, handler EventHandler) {
	if api.eventHandlers[eventType] == nil {
		api.eventHandlers[eventType] = make([]EventHandler, 0)
	}
	api.eventHandlers[eventType] = append(api.eventHandlers[eventType], handler)
}

// emitEvent emits an event to all registered handlers
func (api *CrossPlatformAPI) emitEvent(eventType string, data map[string]interface{}) {
	event := &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	if handlers, exists := api.eventHandlers[eventType]; exists {
		for _, handler := range handlers {
			go handler(event) // Handle asynchronously
		}
	}
}

// Utility APIs

// GetDeviceFingerprint returns a human-readable device fingerprint
func (api *CrossPlatformAPI) GetDeviceFingerprint() string {
	return api.pairingManager.GenerateDeviceFingerprint()
}

// ExportConfig exports device configuration for backup
func (api *CrossPlatformAPI) ExportConfig() ([]byte, error) {
	config := map[string]interface{}{
		"device_id":   api.protocol.GetDeviceID(),
		"device_name": api.pairingManager.GetDeviceName(),
		"version":     "1.0.0",
		"exported_at": time.Now(),
	}
	return json.Marshal(config)
}

// ImportConfig imports device configuration from backup
func (api *CrossPlatformAPI) ImportConfig(configData []byte) error {
	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return err
	}
	
	// This would restore the configuration
	api.emitEvent("config.imported", config)
	return nil
}

// Close shuts down the API
func (api *CrossPlatformAPI) Close() error {
	api.cancel()
	return nil
}
