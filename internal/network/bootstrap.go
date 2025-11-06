package network

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// BootstrapService handles internet-wide device discovery with comprehensive error handling
type BootstrapService struct {
	nodes      []string
	client     *http.Client
	timeout    time.Duration
	retryCount int
	retryDelay time.Duration
	dht        *DHTService
	mu         sync.RWMutex
	stats      BootstrapStats
}

// BootstrapStats tracks service performance
type BootstrapStats struct {
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedRequests  int64
	AvgResponseTime time.Duration
	ActiveNodes     int
	DHTEnabled      bool
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

// NewBootstrapService creates a new bootstrap service with production settings
func NewBootstrapService() *BootstrapService {
	dht := NewDHTService()

	bs := &BootstrapService{
		nodes: []string{
			"https://bootstrap1.fybrk.com",
			"https://bootstrap2.fybrk.com",
			"https://dht.fybrk.com",
		},
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
			},
		},
		timeout:    10 * time.Minute,
		retryCount: 3,
		retryDelay: 2 * time.Second,
		dht:        dht,
	}

	// Start DHT service as fallback
	if err := bs.dht.Start(); err == nil {
		bs.stats.DHTEnabled = true
	}

	return bs
}

// CreateRendezvous creates a temporary rendezvous point with comprehensive error handling
func (bs *BootstrapService) CreateRendezvous(deviceID, publicKey, networkInfo string) (*RendezvousInfo, error) {
	start := time.Now()
	bs.mu.Lock()
	bs.stats.TotalRequests++
	bs.mu.Unlock()

	defer func() {
		bs.mu.Lock()
		bs.stats.AvgResponseTime = time.Since(start)
		bs.mu.Unlock()
	}()

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

	// Try bootstrap nodes first
	var lastErr error
	for _, node := range bs.nodes {
		if err := bs.registerRendezvousWithRetry(node, rendezvous); err == nil {
			bs.mu.Lock()
			bs.stats.SuccessfulReqs++
			bs.mu.Unlock()
			return rendezvous, nil
		} else {
			lastErr = err
		}
	}

	// Fallback to DHT if bootstrap nodes fail
	if bs.stats.DHTEnabled {
		deviceInfo := map[string]interface{}{
			"device_id":    deviceID,
			"public_key":   publicKey,
			"network_info": networkInfo,
			"expires_at":   rendezvous.ExpiresAt.Unix(),
		}

		if err := bs.dht.CreateRendezvous(rendezvous.ID, deviceInfo); err == nil {
			bs.mu.Lock()
			bs.stats.SuccessfulReqs++
			bs.mu.Unlock()
			return rendezvous, nil
		}
	}

	bs.mu.Lock()
	bs.stats.FailedRequests++
	bs.mu.Unlock()

	return nil, fmt.Errorf("failed to register with any service: %v", lastErr)
}

// FindRendezvous looks up a rendezvous point with fallback mechanisms
func (bs *BootstrapService) FindRendezvous(rendezvousID string) (*RendezvousInfo, error) {
	start := time.Now()
	bs.mu.Lock()
	bs.stats.TotalRequests++
	bs.mu.Unlock()

	defer func() {
		bs.mu.Lock()
		bs.stats.AvgResponseTime = time.Since(start)
		bs.mu.Unlock()
	}()

	// Try bootstrap nodes first
	for _, node := range bs.nodes {
		if info, err := bs.lookupRendezvousWithRetry(node, rendezvousID); err == nil {
			bs.mu.Lock()
			bs.stats.SuccessfulReqs++
			bs.mu.Unlock()
			return info, nil
		}
	}

	// Fallback to DHT
	if bs.stats.DHTEnabled {
		if addrs, err := bs.dht.FindRendezvous(rendezvousID); err == nil && len(addrs) > 0 {
			// Create mock rendezvous info from DHT data
			info := &RendezvousInfo{
				ID:          rendezvousID,
				NetworkInfo: addrs[0],                        // DHT returns strings now
				ExpiresAt:   time.Now().Add(5 * time.Minute), // Assume 5min expiry
				Created:     time.Now().Add(-5 * time.Minute),
			}

			bs.mu.Lock()
			bs.stats.SuccessfulReqs++
			bs.mu.Unlock()
			return info, nil
		}
	}

	bs.mu.Lock()
	bs.stats.FailedRequests++
	bs.mu.Unlock()

	return nil, fmt.Errorf("rendezvous not found or expired")
}

// registerRendezvousWithRetry registers with retry logic and exponential backoff
func (bs *BootstrapService) registerRendezvousWithRetry(nodeURL string, info *RendezvousInfo) error {
	var lastErr error

	for attempt := 0; attempt < bs.retryCount; attempt++ {
		if attempt > 0 {
			delay := bs.retryDelay * time.Duration(1<<(attempt-1)) // Exponential backoff
			time.Sleep(delay)
		}

		if err := bs.registerRendezvous(nodeURL, info); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("failed after %d attempts: %v", bs.retryCount, lastErr)
}

// lookupRendezvousWithRetry looks up with retry logic
func (bs *BootstrapService) lookupRendezvousWithRetry(nodeURL, rendezvousID string) (*RendezvousInfo, error) {
	var lastErr error

	for attempt := 0; attempt < bs.retryCount; attempt++ {
		if attempt > 0 {
			delay := bs.retryDelay * time.Duration(1<<(attempt-1))
			time.Sleep(delay)
		}

		if info, err := bs.lookupRendezvous(nodeURL, rendezvousID); err == nil {
			return info, nil
		} else {
			lastErr = err
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", bs.retryCount, lastErr)
}

// registerRendezvous registers a rendezvous with a bootstrap node
func (bs *BootstrapService) registerRendezvous(nodeURL string, info *RendezvousInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", nodeURL+"/rendezvous", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Fybrk/1.0")

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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", nodeURL+"/rendezvous/"+rendezvousID, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Fybrk/1.0")

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

// DiscoverPeers discovers peers using multiple methods
func (bs *BootstrapService) DiscoverPeers(deviceID string) ([]string, error) {
	peers := []string{}

	// Try bootstrap nodes
	for _, node := range bs.nodes {
		if nodePeers, err := bs.queryPeers(node, deviceID); err == nil {
			peers = append(peers, nodePeers...)
		}
	}

	// Try DHT
	if bs.stats.DHTEnabled {
		if addrs, err := bs.dht.FindPeers(deviceID); err == nil {
			peers = append(peers, addrs...)
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

	req.Header.Set("User-Agent", "Fybrk/1.0")

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

// GetStats returns service statistics
func (bs *BootstrapService) GetStats() BootstrapStats {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	stats := bs.stats
	stats.ActiveNodes = len(bs.nodes)
	return stats
}

// Close shuts down the bootstrap service
func (bs *BootstrapService) Close() error {
	if bs.dht != nil {
		return bs.dht.Stop()
	}
	return nil
}
