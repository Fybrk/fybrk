package network

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BootstrapService handles internet-wide device discovery
type BootstrapService struct {
	nodes   []string
	client  *http.Client
	timeout time.Duration
}

// RendezvousInfo contains temporary pairing information
type RendezvousInfo struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	PublicKey   string    `json:"public_key"`
	NetworkInfo string    `json:"network_info"`
	ExpiresAt   time.Time `json:"expires_at"`
	Created     time.Time `json:"created"`
}

// NewBootstrapService creates a new bootstrap service
func NewBootstrapService() *BootstrapService {
	return &BootstrapService{
		nodes: []string{
			"https://bootstrap1.fybrk.com",
			"https://bootstrap2.fybrk.com",
			"https://dht.fybrk.com",
			// Fallback to public DHT nodes
			"https://router.bittorrent.com:6881",
		},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeout: 5 * time.Minute, // Rendezvous expires after 5 minutes
	}
}

// CreateRendezvous creates a temporary rendezvous point for device pairing
func (bs *BootstrapService) CreateRendezvous(deviceID, publicKey, networkInfo string) (*RendezvousInfo, error) {
	// Generate random rendezvous ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("failed to generate rendezvous ID: %v", err)
	}
	
	rendezvous := &RendezvousInfo{
		ID:          hex.EncodeToString(idBytes),
		DeviceID:    deviceID,
		PublicKey:   publicKey,
		NetworkInfo: networkInfo,
		ExpiresAt:   time.Now().Add(bs.timeout),
		Created:     time.Now(),
	}
	
	// Try to register with bootstrap nodes
	for _, node := range bs.nodes {
		if err := bs.registerRendezvous(node, rendezvous); err == nil {
			return rendezvous, nil
		}
	}
	
	return nil, fmt.Errorf("failed to register with any bootstrap node")
}

// FindRendezvous looks up a rendezvous point by ID
func (bs *BootstrapService) FindRendezvous(rendezvousID string) (*RendezvousInfo, error) {
	for _, node := range bs.nodes {
		if info, err := bs.lookupRendezvous(node, rendezvousID); err == nil {
			return info, nil
		}
	}
	
	return nil, fmt.Errorf("rendezvous not found or expired")
}

// registerRendezvous registers a rendezvous with a bootstrap node
func (bs *BootstrapService) registerRendezvous(nodeURL string, info *RendezvousInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "POST", nodeURL+"/rendezvous", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := bs.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bootstrap node returned status %d", resp.StatusCode)
	}
	
	return nil
}

// lookupRendezvous looks up a rendezvous from a bootstrap node
func (bs *BootstrapService) lookupRendezvous(nodeURL, rendezvousID string) (*RendezvousInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", nodeURL+"/rendezvous/"+rendezvousID, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := bs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rendezvous not found")
	}
	
	var info RendezvousInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	
	// Check if expired
	if time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("rendezvous expired")
	}
	
	return &info, nil
}

// DiscoverPeers discovers peers using DHT-like distributed hash table
func (bs *BootstrapService) DiscoverPeers(deviceID string) ([]string, error) {
	// Simple peer discovery - in production this would use a proper DHT
	peers := []string{}
	
	for _, node := range bs.nodes {
		if nodePeers, err := bs.queryPeers(node, deviceID); err == nil {
			peers = append(peers, nodePeers...)
		}
	}
	
	return peers, nil
}

// queryPeers queries a bootstrap node for known peers
func (bs *BootstrapService) queryPeers(nodeURL, deviceID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", nodeURL+"/peers?device="+deviceID, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := bs.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query peers")
	}
	
	var peers []string
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, err
	}
	
	return peers, nil
}
