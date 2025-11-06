package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkManager(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)
	require.NoError(t, err)
	defer fybrk.Close()

	syncEngine, err := NewSyncEngine(fybrk)
	require.NoError(t, err)
	defer syncEngine.Stop()

	nm := NewNetworkManager(fybrk, syncEngine)
	defer nm.Stop()

	assert.NotNil(t, nm)
	assert.NotNil(t, nm.fybrk)
	assert.NotNil(t, nm.syncEngine)
	assert.Equal(t, 8080, nm.port)
}

func TestNetworkManagerStartServer(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)
	require.NoError(t, err)
	defer fybrk.Close()

	syncEngine, err := NewSyncEngine(fybrk)
	require.NoError(t, err)
	defer syncEngine.Stop()

	nm := NewNetworkManager(fybrk, syncEngine)
	defer nm.Stop()

	// Start server
	err = nm.StartServer()
	assert.NoError(t, err)

	// Should have a server running
	assert.NotNil(t, nm.server)
}

func TestNetworkManagerConnectToPeer(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)
	require.NoError(t, err)
	defer fybrk.Close()

	syncEngine, err := NewSyncEngine(fybrk)
	require.NoError(t, err)
	defer syncEngine.Stop()

	nm := NewNetworkManager(fybrk, syncEngine)
	defer nm.Stop()

	// Try to connect to invalid address
	err = nm.ConnectToPeer("invalid-address:99999")
	assert.Error(t, err)
}

func TestNetworkManagerGetLocalAddress(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)
	require.NoError(t, err)
	defer fybrk.Close()

	syncEngine, err := NewSyncEngine(fybrk)
	require.NoError(t, err)
	defer syncEngine.Stop()

	nm := NewNetworkManager(fybrk, syncEngine)
	defer nm.Stop()

	// Start server first
	err = nm.StartServer()
	require.NoError(t, err)

	addr := nm.GetLocalAddress()
	assert.NotEmpty(t, addr)
	assert.Contains(t, addr, ":")
}

func TestNetworkManagerStop(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{SyncPath: tempDir}
	fybrk, err := New(config)
	require.NoError(t, err)
	defer fybrk.Close()

	syncEngine, err := NewSyncEngine(fybrk)
	require.NoError(t, err)
	defer syncEngine.Stop()

	nm := NewNetworkManager(fybrk, syncEngine)

	// Start server
	err = nm.StartServer()
	require.NoError(t, err)

	// Stop should work without error
	err = nm.Stop()
	assert.NoError(t, err)
}
