package pairing

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDevicePairingManager(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	assert.Equal(t, "device-123", dpm.deviceID)
	assert.Equal(t, "Test Device", dpm.deviceName)
	assert.Equal(t, "Test Device", dpm.GetDeviceName())
}

func TestGeneratePairingQR(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	qrImage, challenge, err := dpm.GeneratePairingQR("192.168.1.100:8080")
	require.NoError(t, err)

	assert.NotEmpty(t, qrImage)
	assert.NotEmpty(t, challenge)
	assert.Greater(t, len(qrImage), 100) // QR image should be substantial
}

func TestParsePairingQR(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	// Create test QR data
	qrData := QRPairingData{
		Version:     1,
		DeviceID:    "device-456",
		DeviceName:  "Remote Device",
		NetworkAddr: "192.168.1.200:8080",
		PublicKey:   "test-public-key",
		Challenge:   "test-challenge",
		Expires:     time.Now().Add(5 * time.Minute).Unix(),
	}

	jsonData, err := json.Marshal(qrData)
	require.NoError(t, err)

	// Parse the QR data
	parsed, err := dpm.ParsePairingQR(string(jsonData))
	require.NoError(t, err)

	assert.Equal(t, qrData.DeviceID, parsed.DeviceID)
	assert.Equal(t, qrData.DeviceName, parsed.DeviceName)
	assert.Equal(t, qrData.NetworkAddr, parsed.NetworkAddr)
}

func TestParsePairingQRExpired(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	// Create expired QR data
	qrData := QRPairingData{
		Version:     1,
		DeviceID:    "device-456",
		DeviceName:  "Remote Device",
		NetworkAddr: "192.168.1.200:8080",
		PublicKey:   "test-public-key",
		Challenge:   "test-challenge",
		Expires:     time.Now().Add(-1 * time.Minute).Unix(), // Expired
	}

	jsonData, err := json.Marshal(qrData)
	require.NoError(t, err)

	// Should fail to parse expired QR
	_, err = dpm.ParsePairingQR(string(jsonData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestInitiatePairing(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	// Set up pairing callback
	var pairedDevice *PairedDevice
	dpm.SetPairingCallback(func(device *PairedDevice) error {
		pairedDevice = device
		return nil
	})

	qrData := &QRPairingData{
		Version:     1,
		DeviceID:    "device-456",
		DeviceName:  "Remote Device",
		NetworkAddr: "192.168.1.200:8080",
		PublicKey:   "test-public-key",
		Challenge:   "test-challenge",
		Expires:     time.Now().Add(5 * time.Minute).Unix(),
	}

	device, err := dpm.InitiatePairing(qrData)
	require.NoError(t, err)

	assert.Equal(t, qrData.DeviceID, device.ID)
	assert.Equal(t, qrData.DeviceName, device.Name)
	assert.Equal(t, 1, device.TrustLevel)
	assert.NotNil(t, pairedDevice)
	assert.Equal(t, device.ID, pairedDevice.ID)
}

func TestGenerateDeviceFingerprint(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	fingerprint := dpm.GenerateDeviceFingerprint()
	assert.NotEmpty(t, fingerprint)
	assert.Greater(t, len(fingerprint), 8) // Should be substantial

	// Generate another fingerprint with different device - should be different
	dpm2 := NewDevicePairingManager("device-456", "Different Device")
	fingerprint2 := dpm2.GenerateDeviceFingerprint()
	assert.NotEqual(t, fingerprint, fingerprint2)
}

func TestGetCapabilities(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	capabilities := dpm.GetCapabilities()
	assert.True(t, capabilities.CanSync)
	assert.True(t, capabilities.CanRelay)
	assert.True(t, capabilities.CanStore)
	assert.False(t, capabilities.HasGUI) // CLI version
	assert.False(t, capabilities.IsMobile)
	assert.False(t, capabilities.IsServer)
}

func TestQRPairingDataValidation(t *testing.T) {
	dpm := NewDevicePairingManager("device-123", "Test Device")

	// Test invalid JSON
	_, err := dpm.ParsePairingQR("invalid json")
	assert.Error(t, err)

	// Test unsupported version
	qrData := QRPairingData{
		Version: 999, // Unsupported version
		Expires: time.Now().Add(5 * time.Minute).Unix(),
	}
	jsonData, _ := json.Marshal(qrData)

	_, err = dpm.ParsePairingQR(string(jsonData))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
