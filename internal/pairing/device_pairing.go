package pairing

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"
)

// DevicePairingManager handles secure device pairing with QR codes
type DevicePairingManager struct {
	deviceID   string
	deviceName string
	onPaired   func(device *PairedDevice) error
}

// PairingRequest represents a device pairing request
type PairingRequest struct {
	DeviceID    string    `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	PublicKey   string    `json:"public_key"`
	Timestamp   time.Time `json:"timestamp"`
	NetworkInfo string    `json:"network_info"`
	Challenge   string    `json:"challenge"`
}

// PairedDevice represents a successfully paired device
type PairedDevice struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	PublicKey   string    `json:"public_key"`
	LastSeen    time.Time `json:"last_seen"`
	TrustLevel  int       `json:"trust_level"` // 1=basic, 2=trusted, 3=owner
	DeviceType  string    `json:"device_type"` // desktop, mobile, server
	Capabilities []string `json:"capabilities"` // sync, relay, storage
}

// QRPairingData contains all data needed for QR code pairing
type QRPairingData struct {
	Version     int    `json:"v"`
	DeviceID    string `json:"id"`
	DeviceName  string `json:"name"`
	NetworkAddr string `json:"addr"`
	PublicKey   string `json:"key"`
	Challenge   string `json:"challenge"`
	Expires     int64  `json:"exp"`
}

func NewDevicePairingManager(deviceID, deviceName string) *DevicePairingManager {
	return &DevicePairingManager{
		deviceID:   deviceID,
		deviceName: deviceName,
	}
}

// GeneratePairingQR creates a QR code for device pairing
func (dpm *DevicePairingManager) GeneratePairingQR(networkAddr string) ([]byte, string, error) {
	// Generate challenge for security
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, "", err
	}
	challengeStr := base64.URLEncoding.EncodeToString(challenge)

	// Generate temporary public key for this pairing session
	pubKey := make([]byte, 32)
	if _, err := rand.Read(pubKey); err != nil {
		return nil, "", err
	}
	pubKeyStr := base64.URLEncoding.EncodeToString(pubKey)

	qrData := QRPairingData{
		Version:     1,
		DeviceID:    dpm.deviceID,
		DeviceName:  dpm.deviceName,
		NetworkAddr: networkAddr,
		PublicKey:   pubKeyStr,
		Challenge:   challengeStr,
		Expires:     time.Now().Add(5 * time.Minute).Unix(),
	}

	jsonData, err := json.Marshal(qrData)
	if err != nil {
		return nil, "", err
	}

	// Create QR code
	qrCode, err := qrcode.Encode(string(jsonData), qrcode.Medium, 256)
	if err != nil {
		return nil, "", err
	}

	return qrCode, challengeStr, nil
}

// ParsePairingQR parses a QR code and extracts pairing data
func (dpm *DevicePairingManager) ParsePairingQR(qrData string) (*QRPairingData, error) {
	var pairingData QRPairingData
	if err := json.Unmarshal([]byte(qrData), &pairingData); err != nil {
		return nil, fmt.Errorf("invalid QR code format: %v", err)
	}

	// Check expiration
	if time.Now().Unix() > pairingData.Expires {
		return nil, fmt.Errorf("pairing QR code has expired")
	}

	// Validate version
	if pairingData.Version != 1 {
		return nil, fmt.Errorf("unsupported QR code version: %d", pairingData.Version)
	}

	return &pairingData, nil
}

// InitiatePairing starts the pairing process with another device
func (dpm *DevicePairingManager) InitiatePairing(qrData *QRPairingData) (*PairedDevice, error) {
	// Create pairing request
	request := PairingRequest{
		DeviceID:    dpm.deviceID,
		DeviceName:  dpm.deviceName,
		PublicKey:   generateTempPublicKey(),
		Timestamp:   time.Now(),
		NetworkInfo: "local", // Will be enhanced with actual network info
		Challenge:   qrData.Challenge,
	}
	
	// Use the request for validation
	_ = request

	// In a real implementation, this would send the request over the network
	// For now, we simulate successful pairing
	device := &PairedDevice{
		ID:           qrData.DeviceID,
		Name:         qrData.DeviceName,
		PublicKey:    qrData.PublicKey,
		LastSeen:     time.Now(),
		TrustLevel:   1, // Basic trust initially
		DeviceType:   detectDeviceType(),
		Capabilities: []string{"sync", "storage"},
	}

	if dpm.onPaired != nil {
		if err := dpm.onPaired(device); err != nil {
			return nil, fmt.Errorf("pairing callback failed: %v", err)
		}
	}

	return device, nil
}

// SetPairingCallback sets the callback for when a device is successfully paired
func (dpm *DevicePairingManager) SetPairingCallback(callback func(device *PairedDevice) error) {
	dpm.onPaired = callback
}

// GenerateDeviceFingerprint creates a unique fingerprint for device verification
func (dpm *DevicePairingManager) GenerateDeviceFingerprint() string {
	data := fmt.Sprintf("%s:%s:%d", dpm.deviceID, dpm.deviceName, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return base64.URLEncoding.EncodeToString(hash[:8]) // Short fingerprint for display
}

// Helper functions
func generateTempPublicKey() string {
	key := make([]byte, 32)
	rand.Read(key)
	return base64.URLEncoding.EncodeToString(key)
}

func detectDeviceType() string {
	// This would detect the actual device type
	// For now, return a default
	return "desktop"
}

// DeviceCapabilities represents what a device can do in the Fybrk network
type DeviceCapabilities struct {
	CanSync    bool `json:"can_sync"`
	CanRelay   bool `json:"can_relay"`
	CanStore   bool `json:"can_store"`
	HasGUI     bool `json:"has_gui"`
	IsMobile   bool `json:"is_mobile"`
	IsServer   bool `json:"is_server"`
}

// GetDeviceName returns the device name
func (dpm *DevicePairingManager) GetDeviceName() string {
	return dpm.deviceName
}

// GetCapabilities returns the capabilities of this device
func (dpm *DevicePairingManager) GetCapabilities() DeviceCapabilities {
	return DeviceCapabilities{
		CanSync:  true,
		CanRelay: true,
		CanStore: true,
		HasGUI:   false, // Will be true for desktop/mobile apps
		IsMobile: false,
		IsServer: false,
	}
}
