package types

import (
	"time"
)

// Chunk represents an encrypted file chunk
type Chunk struct {
	Hash      [32]byte  `json:"hash"`
	Data      []byte    `json:"data"`
	Size      int64     `json:"size"`
	Encrypted bool      `json:"encrypted"`
	CreatedAt time.Time `json:"created_at"`
}

// FileMetadata represents file information in the sync system
type FileMetadata struct {
	Path    string     `json:"path"`
	Hash    [32]byte   `json:"hash"`
	Size    int64      `json:"size"`
	ModTime time.Time  `json:"mod_time"`
	Chunks  [][32]byte `json:"chunks"`
	Version int64      `json:"version"`
}

// DeviceProfile defines how a device handles data
type DeviceProfile int

const (
	FullReplica DeviceProfile = iota // Store all data
	SmartCache                       // Cache recent files
	IndexOnly                        // Metadata only
)

// Device represents a node in the Fybrk network
type Device struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Profile  DeviceProfile `json:"profile"`
	LastSeen time.Time     `json:"last_seen"`
}

// SyncEvent represents a synchronization event
type SyncEvent struct {
	Type      string    `json:"type"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	DeviceID  string    `json:"device_id"`
}
