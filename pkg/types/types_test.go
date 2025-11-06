package types

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChunk(t *testing.T) {
	data := []byte("test data")
	hash := sha256.Sum256(data)

	chunk := Chunk{
		Hash:      hash,
		Data:      data,
		Size:      int64(len(data)),
		Encrypted: false,
		CreatedAt: time.Now(),
	}

	assert.Equal(t, hash, chunk.Hash)
	assert.Equal(t, data, chunk.Data)
	assert.Equal(t, int64(9), chunk.Size)
	assert.False(t, chunk.Encrypted)
}

func TestFileMetadata(t *testing.T) {
	path := "/test/file.txt"
	hash := sha256.Sum256([]byte("content"))
	chunks := [][32]byte{hash}

	metadata := FileMetadata{
		Path:    path,
		Hash:    hash,
		Size:    100,
		ModTime: time.Now(),
		Chunks:  chunks,
		Version: 1,
	}

	assert.Equal(t, path, metadata.Path)
	assert.Equal(t, hash, metadata.Hash)
	assert.Equal(t, int64(100), metadata.Size)
	assert.Equal(t, chunks, metadata.Chunks)
	assert.Equal(t, int64(1), metadata.Version)
}

func TestDeviceProfile(t *testing.T) {
	assert.Equal(t, DeviceProfile(0), FullReplica)
	assert.Equal(t, DeviceProfile(1), SmartCache)
	assert.Equal(t, DeviceProfile(2), IndexOnly)
}

func TestDevice(t *testing.T) {
	device := Device{
		ID:       "device-123",
		Name:     "Test Device",
		Profile:  FullReplica,
		LastSeen: time.Now(),
	}

	assert.Equal(t, "device-123", device.ID)
	assert.Equal(t, "Test Device", device.Name)
	assert.Equal(t, FullReplica, device.Profile)
}
