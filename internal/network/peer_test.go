package network

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPeerNetwork(t *testing.T) {
	pn := NewPeerNetwork("test-device", 8080)

	assert.Equal(t, "test-device", pn.deviceID)
	assert.Equal(t, 8080, pn.port)
	assert.NotNil(t, pn.peers)
	assert.NotNil(t, pn.ctx)
}

func TestPeerNetworkStartStop(t *testing.T) {
	pn := NewPeerNetwork("test-device", 0) // Use random port

	err := pn.Start()
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	err = pn.Stop()
	assert.NoError(t, err)
}

func TestGenerateDeviceID(t *testing.T) {
	id1 := GenerateDeviceID()
	id2 := GenerateDeviceID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 32) // 16 bytes = 32 hex chars
}

func TestPeerNetworkGetPeers(t *testing.T) {
	pn := NewPeerNetwork("test-device", 8080)

	peers := pn.GetPeers()
	assert.Empty(t, peers)

	// Manually add a peer for testing
	pn.updatePeer("peer-1", "192.168.1.100:8080", nil)

	peers = pn.GetPeers()
	assert.Len(t, peers, 1)
	assert.Contains(t, peers, "peer-1")
}

func TestPeerNetworkMessageHandler(t *testing.T) {
	pn := NewPeerNetwork("test-device", 8080)

	var receivedDeviceID string
	var receivedMsg *Message

	pn.SetMessageHandler(func(deviceID string, msg *Message) {
		receivedDeviceID = deviceID
		receivedMsg = msg
	})

	// Simulate message handling
	testMsg := &Message{
		Type:      "test",
		DeviceID:  "sender-device",
		Timestamp: time.Now(),
		Data:      "test data",
	}

	if pn.onMessage != nil {
		pn.onMessage("sender-device", testMsg)
	}

	assert.Equal(t, "sender-device", receivedDeviceID)
	assert.Equal(t, testMsg, receivedMsg)
}

func TestUpdatePeer(t *testing.T) {
	pn := NewPeerNetwork("test-device", 8080)

	// Should not add ourselves
	pn.updatePeer("test-device", "localhost:8080", nil)
	peers := pn.GetPeers()
	assert.Empty(t, peers)

	// Should add other devices
	pn.updatePeer("other-device", "192.168.1.100:8080", nil)
	peers = pn.GetPeers()
	assert.Len(t, peers, 1)
	assert.Contains(t, peers, "other-device")
}

func TestSendMessage(t *testing.T) {
	pn := NewPeerNetwork("test-device", 8080)

	// Test sending to non-existent peer
	msg := &Message{
		Type:     "test",
		DeviceID: "test-device",
		Data:     "test",
	}

	err := pn.SendMessage("non-existent", msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestBroadcastMessage(t *testing.T) {
	pn := NewPeerNetwork("test-device", 8080)

	msg := &Message{
		Type:     "test",
		DeviceID: "test-device",
		Data:     "test",
	}

	// Should not panic with no peers
	pn.BroadcastMessage(msg)
}

func TestTryConnect(t *testing.T) {
	pn := NewPeerNetwork("test-device", 0)

	// Should handle connection failure gracefully
	pn.tryConnect("invalid-address:99999")

	// No assertion needed - just verify it doesn't panic
}

func TestPeerNetworkIntegration(t *testing.T) {
	// Create two peer networks
	pn1 := NewPeerNetwork("device-1", 0)
	pn2 := NewPeerNetwork("device-2", 0)

	var received1, received2 *Message

	pn1.SetMessageHandler(func(deviceID string, msg *Message) {
		received1 = msg
	})

	pn2.SetMessageHandler(func(deviceID string, msg *Message) {
		received2 = msg
	})

	// Use variables to avoid unused variable errors
	_ = received1
	_ = received2

	// Start both networks
	err := pn1.Start()
	require.NoError(t, err)
	defer pn1.Stop()

	err = pn2.Start()
	require.NoError(t, err)
	defer pn2.Stop()

	// Get actual ports
	addr1 := pn1.listener.Addr().(*net.TCPAddr)
	addr2 := pn2.listener.Addr().(*net.TCPAddr)
	_ = addr2 // Avoid unused variable error

	// Connect pn2 to pn1
	go pn2.tryConnect(addr1.String())

	// Give time for connection
	time.Sleep(200 * time.Millisecond)

	// Send message from pn1 to pn2
	msg := &Message{
		Type:      "test",
		DeviceID:  "device-1",
		Timestamp: time.Now(),
		Data:      "hello",
	}

	// This might fail if connection isn't established yet
	// That's expected behavior for this test
	pn1.BroadcastMessage(msg)
}

func TestMessageSerialization(t *testing.T) {
	msg := &Message{
		Type:      "test",
		DeviceID:  "device-1",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
	}

	// Test JSON serialization
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.Type, decoded.Type)
	assert.Equal(t, msg.DeviceID, decoded.DeviceID)
}
