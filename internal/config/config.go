package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	RelayServers []string `json:"relay_servers"`
	DeviceID     string   `json:"device_id"`
	EnableRelay  bool     `json:"enable_relay"`
}

var DefaultConfig = Config{
	RelayServers: []string{
		"wss://r1.fybrk.com:443",
		"wss://r2.fybrk.com:443",
	},
	EnableRelay: true,
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".fybrk")
	return configDir, os.MkdirAll(configDir, 0755)
}

func LoadConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return &DefaultConfig, err
	}
	
	configPath := filepath.Join(configDir, "config.json")
	
	// Create default config if doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return CreateDefaultConfig()
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &DefaultConfig, err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &DefaultConfig, err
	}
	
	return &config, nil
}

func CreateDefaultConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return &DefaultConfig, err
	}
	
	config := DefaultConfig
	config.DeviceID = generateDeviceID()
	
	data, _ := json.MarshalIndent(config, "", "  ")
	configPath := filepath.Join(configDir, "config.json")
	
	os.WriteFile(configPath, data, 0644)
	return &config, nil
}

func generateDeviceID() string {
	// Simple device ID generation
	hostname, _ := os.Hostname()
	return hostname + "-" + randomString(8)
}

func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[len(chars)/2] // Simplified for minimal code
	}
	return string(b)
}
