package network

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Fybrk/fybrk/pkg/types"
)

type PeerNetwork struct {
	deviceID    string
	port        int
	peers       map[string]*Peer
	listener    net.Listener
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	onMessage   func(deviceID string, msg *Message)
	upnp        *UPnPClient // UPnP client for NAT traversal
	bootstrap   *BootstrapService // Bootstrap service for internet discovery
	holePuncher *HolePuncher // Hole puncher for NAT traversal
}

type Peer struct {
	DeviceID string
	Address  string
	Conn     net.Conn
	LastSeen time.Time
}

type Message struct {
	Type      string      `json:"type"`
	DeviceID  string      `json:"device_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type FileRequest struct {
	Path   string     `json:"path"`
	Chunks [][32]byte `json:"chunks"`
}

type FileResponse struct {
	Path   string           `json:"path"`
	Chunks []types.Chunk    `json:"chunks"`
}

func NewPeerNetwork(deviceID string, port int) *PeerNetwork {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PeerNetwork{
		deviceID:    deviceID,
		port:        port,
		peers:       make(map[string]*Peer),
		ctx:         ctx,
		cancel:      cancel,
		bootstrap:   NewBootstrapService(),
		holePuncher: NewHolePuncher(),
	}
}

func (pn *PeerNetwork) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", pn.port))
	if err != nil {
		return err
	}
	
	pn.listener = listener
	
	// Try to set up UPnP port forwarding (optional, don't fail if unavailable)
	if upnp, err := NewUPnPClient(); err == nil {
		pn.upnp = upnp
		actualPort := pn.listener.Addr().(*net.TCPAddr).Port
		if err := upnp.AddPortMapping(actualPort, actualPort, "TCP"); err == nil {
			fmt.Printf("UPnP port forwarding enabled on port %d\n", actualPort)
		}
	}
	
	go pn.acceptConnections()
	go pn.discoverPeers()
	
	return nil
}

func (pn *PeerNetwork) Stop() error {
	pn.cancel()
	
	// Clean up UPnP port forwarding
	if pn.upnp != nil && pn.listener != nil {
		actualPort := pn.listener.Addr().(*net.TCPAddr).Port
		pn.upnp.RemovePortMapping(actualPort, "TCP")
	}
	
	if pn.listener != nil {
		pn.listener.Close()
	}
	
	pn.mu.Lock()
	for _, peer := range pn.peers {
		if peer.Conn != nil {
			peer.Conn.Close()
		}
	}
	pn.mu.Unlock()
	
	return nil
}

func (pn *PeerNetwork) acceptConnections() {
	for {
		select {
		case <-pn.ctx.Done():
			return
		default:
			conn, err := pn.listener.Accept()
			if err != nil {
				continue
			}
			
			go pn.handleConnection(conn)
		}
	}
}

func (pn *PeerNetwork) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	decoder := json.NewDecoder(conn)
	
	for {
		select {
		case <-pn.ctx.Done():
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				return
			}
			
			pn.updatePeer(msg.DeviceID, conn.RemoteAddr().String(), conn)
			
			if pn.onMessage != nil {
				pn.onMessage(msg.DeviceID, &msg)
			}
		}
	}
}

func (pn *PeerNetwork) discoverPeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-pn.ctx.Done():
			return
		case <-ticker.C:
			// Try local network discovery first
			pn.broadcastDiscovery()
			
			// Try internet discovery via bootstrap
			pn.discoverInternetPeers()
		}
	}
}

func (pn *PeerNetwork) discoverInternetPeers() {
	// Discover peers via bootstrap network
	peers, err := pn.bootstrap.DiscoverPeers(pn.deviceID)
	if err != nil {
		return // Silently fail - local discovery might still work
	}
	
	// Try to connect to discovered peers
	for _, peerAddr := range peers {
		go pn.tryConnectWithHolePunch(peerAddr)
	}
}

func (pn *PeerNetwork) tryConnectWithHolePunch(addr string) {
	// First try direct connection
	pn.tryConnect(addr)
	
	// If direct connection fails, try hole punching
	// This would parse the address and attempt hole punching
	// Implementation simplified for now
}

func (pn *PeerNetwork) broadcastDiscovery() {
	// Simple broadcast on local network
	for i := 1; i < 255; i++ {
		addr := fmt.Sprintf("192.168.1.%d:%d", i, pn.port)
		go pn.tryConnect(addr)
	}
}

func (pn *PeerNetwork) tryConnect(addr string) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return
	}
	
	// Send discovery message
	msg := Message{
		Type:      "discovery",
		DeviceID:  pn.deviceID,
		Timestamp: time.Now(),
	}
	
	encoder := json.NewEncoder(conn)
	encoder.Encode(msg)
	
	go pn.handleConnection(conn)
}

func (pn *PeerNetwork) updatePeer(deviceID, address string, conn net.Conn) {
	if deviceID == pn.deviceID {
		return // Don't add ourselves
	}
	
	pn.mu.Lock()
	defer pn.mu.Unlock()
	
	pn.peers[deviceID] = &Peer{
		DeviceID: deviceID,
		Address:  address,
		Conn:     conn,
		LastSeen: time.Now(),
	}
}

func (pn *PeerNetwork) SendMessage(deviceID string, msg *Message) error {
	pn.mu.RLock()
	peer, exists := pn.peers[deviceID]
	pn.mu.RUnlock()
	
	if !exists || peer.Conn == nil {
		return fmt.Errorf("peer %s not connected", deviceID)
	}
	
	encoder := json.NewEncoder(peer.Conn)
	return encoder.Encode(msg)
}

func (pn *PeerNetwork) BroadcastMessage(msg *Message) {
	pn.mu.RLock()
	defer pn.mu.RUnlock()
	
	for _, peer := range pn.peers {
		if peer.Conn != nil {
			encoder := json.NewEncoder(peer.Conn)
			encoder.Encode(msg)
		}
	}
}

func (pn *PeerNetwork) GetPeers() []string {
	pn.mu.RLock()
	defer pn.mu.RUnlock()
	
	var peers []string
	for deviceID := range pn.peers {
		peers = append(peers, deviceID)
	}
	
	return peers
}

func (pn *PeerNetwork) SetMessageHandler(handler func(deviceID string, msg *Message)) {
	pn.onMessage = handler
}

// CreatePairingQR creates a QR code for internet-wide device pairing
func (pn *PeerNetwork) CreatePairingQR(syncPath, encryptionKey string) (string, error) {
	// Get public address via STUN
	stunResp, err := pn.holePuncher.GetPublicAddress()
	if err != nil {
		return "", fmt.Errorf("failed to get public address: %v", err)
	}
	
	// Create rendezvous point
	networkInfo := fmt.Sprintf("%s:%d", stunResp.PublicIP, stunResp.PublicPort)
	rendezvous, err := pn.bootstrap.CreateRendezvous(pn.deviceID, "temp-public-key", networkInfo)
	if err != nil {
		return "", fmt.Errorf("failed to create rendezvous: %v", err)
	}
	
	// Create pairing data
	pairingData := map[string]interface{}{
		"version":       1,
		"rendezvous_id": rendezvous.ID,
		"device_id":     pn.deviceID,
		"sync_path":     syncPath,
		"encryption_key": encryptionKey,
		"expires_at":    rendezvous.ExpiresAt.Unix(),
	}
	
	// Convert to JSON
	jsonData, err := json.Marshal(pairingData)
	if err != nil {
		return "", err
	}
	
	// Create fybrk:// URL
	qrData := fmt.Sprintf("fybrk://pair?data=%s", base64.URLEncoding.EncodeToString(jsonData))
	
	return qrData, nil
}

// JoinFromQR joins a sync network from QR code data
func (pn *PeerNetwork) JoinFromQR(qrData string) error {
	// Parse fybrk:// URL
	if !strings.HasPrefix(qrData, "fybrk://pair?data=") {
		return fmt.Errorf("invalid QR code format")
	}
	
	// Extract and decode data
	encodedData := strings.TrimPrefix(qrData, "fybrk://pair?data=")
	jsonData, err := base64.URLEncoding.DecodeString(encodedData)
	if err != nil {
		return fmt.Errorf("failed to decode QR data: %v", err)
	}
	
	// Parse pairing data
	var pairingData map[string]interface{}
	if err := json.Unmarshal(jsonData, &pairingData); err != nil {
		return fmt.Errorf("failed to parse pairing data: %v", err)
	}
	
	// Extract rendezvous ID
	rendezvousID, ok := pairingData["rendezvous_id"].(string)
	if !ok {
		return fmt.Errorf("missing rendezvous ID in QR data")
	}
	
	// Look up rendezvous
	rendezvous, err := pn.bootstrap.FindRendezvous(rendezvousID)
	if err != nil {
		return fmt.Errorf("failed to find rendezvous: %v", err)
	}
	
	// Connect to the device
	return pn.connectToRendezvous(rendezvous)
}

// connectToRendezvous connects to a device via rendezvous
func (pn *PeerNetwork) connectToRendezvous(rendezvous *RendezvousInfo) error {
	// Try direct connection first
	pn.tryConnect(rendezvous.NetworkInfo)
	
	// TODO: Implement hole punching if direct connection fails
	
	return nil
}

func GenerateDeviceID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
