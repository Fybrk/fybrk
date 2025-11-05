package network

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/skip2/go-qrcode"
)

// QRGenerator handles QR code generation and display
type QRGenerator struct{}

// NewQRGenerator creates a new QR code generator
func NewQRGenerator() *QRGenerator {
	return &QRGenerator{}
}

// GenerateQRCode creates a QR code for pairing data
func (qg *QRGenerator) GenerateQRCode(pairingData map[string]interface{}) (string, error) {
	// Convert to JSON
	jsonData, err := json.Marshal(pairingData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal pairing data: %v", err)
	}

	// Create fybrk:// URL
	qrData := fmt.Sprintf("fybrk://pair?data=%s", base64.URLEncoding.EncodeToString(jsonData))

	return qrData, nil
}

// DisplayQRCode shows QR code in terminal and optionally saves to file
func (qg *QRGenerator) DisplayQRCode(data string, saveToFile bool) error {
	// Generate QR code for terminal display
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %v", err)
	}

	// Display in terminal
	fmt.Println("QR CODE:")
	fmt.Println(qr.ToSmallString(false))
	fmt.Println()
	fmt.Printf("QR Data: %s\n", data)

	// Optionally save to file
	if saveToFile {
		filename := fmt.Sprintf("fybrk-pair-%d.png", time.Now().Unix())
		err = qr.WriteFile(256, filename)
		if err != nil {
			fmt.Printf("Warning: Could not save QR code to file: %v\n", err)
		} else {
			fmt.Printf("QR code saved to: %s\n", filename)
		}
	}

	return nil
}

// ParseQRData extracts pairing data from QR code
func (qg *QRGenerator) ParseQRData(qrData string) (map[string]interface{}, error) {
	if len(qrData) < 20 || qrData[:17] != "fybrk://pair?data=" {
		return nil, fmt.Errorf("invalid QR code format")
	}

	// Extract and decode data
	encodedData := qrData[17:]
	jsonData, err := base64.URLEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode QR data: %v", err)
	}

	// Parse pairing data
	var pairingData map[string]interface{}
	if err := json.Unmarshal(jsonData, &pairingData); err != nil {
		return nil, fmt.Errorf("failed to parse pairing data: %v", err)
	}

	// Validate expiration
	if expiresAt, ok := pairingData["expires_at"].(float64); ok {
		if time.Now().Unix() > int64(expiresAt) {
			return nil, fmt.Errorf("QR code has expired")
		}
	}

	return pairingData, nil
}

// ScanQRFromFile reads QR code from image file
func (qg *QRGenerator) ScanQRFromFile(filename string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", fmt.Errorf("QR code file not found: %s", filename)
	}

	// Note: For full QR scanning from images, we'd need a QR decoder library
	// For now, return an error suggesting manual input
	return "", fmt.Errorf("QR code scanning from files not yet implemented. Please copy the QR data manually")
}
