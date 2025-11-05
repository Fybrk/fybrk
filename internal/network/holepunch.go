package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// HolePuncher handles NAT traversal using hole punching techniques
type HolePuncher struct {
	localAddr  *net.UDPAddr
	conn       *net.UDPConn
	stunServer string
	mu         sync.RWMutex
}

// STUNResponse contains STUN server response with public IP/port
type STUNResponse struct {
	PublicIP   net.IP
	PublicPort int
}

// NewHolePuncher creates a new hole puncher
func NewHolePuncher() *HolePuncher {
	return &HolePuncher{
		stunServer: "stun.l.google.com:19302", // Free Google STUN server
	}
}

// GetPublicAddress discovers public IP and port using STUN
func (hp *HolePuncher) GetPublicAddress() (*STUNResponse, error) {
	// Create UDP connection
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %v", err)
	}
	defer conn.Close()
	
	hp.conn = conn
	hp.localAddr = conn.LocalAddr().(*net.UDPAddr)
	
	// Resolve STUN server
	stunAddr, err := net.ResolveUDPAddr("udp", hp.stunServer)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server: %v", err)
	}
	
	// Send STUN binding request
	stunRequest := hp.createSTUNRequest()
	_, err = conn.WriteToUDP(stunRequest, stunAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %v", err)
	}
	
	// Read STUN response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read STUN response: %v", err)
	}
	
	// Parse STUN response
	return hp.parseSTUNResponse(buffer[:n])
}

// PunchHole attempts to establish direct connection with peer
func (hp *HolePuncher) PunchHole(peerPublicIP net.IP, peerPublicPort int, peerPrivateIP net.IP, peerPrivatePort int) (*net.UDPConn, error) {
	// Create connection for hole punching
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create hole punch connection: %v", err)
	}
	
	// Try multiple connection attempts
	attempts := []net.UDPAddr{
		{IP: peerPublicIP, Port: peerPublicPort},   // Try public address first
		{IP: peerPrivateIP, Port: peerPrivatePort}, // Try private address (same network)
	}
	
	// Send hole punch packets to all possible addresses
	punchData := []byte("FYBRK_HOLE_PUNCH")
	
	for i := 0; i < 10; i++ { // Multiple attempts
		for _, addr := range attempts {
			conn.WriteToUDP(punchData, &addr)
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	// Listen for response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	n, peerAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("hole punch failed: %v", err)
	}
	
	// Verify it's a valid response
	if string(buffer[:n]) == "FYBRK_HOLE_PUNCH_ACK" {
		fmt.Printf("Hole punch successful to %s\n", peerAddr)
		return conn, nil
	}
	
	conn.Close()
	return nil, fmt.Errorf("invalid hole punch response")
}

// StartHolePunchListener listens for incoming hole punch attempts
func (hp *HolePuncher) StartHolePunchListener(ctx context.Context, port int) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	fmt.Printf("Hole punch listener started on port %d\n", port)
	
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			buffer := make([]byte, 1024)
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				continue // Timeout or other error, keep listening
			}
			
			// Check if it's a hole punch request
			if string(buffer[:n]) == "FYBRK_HOLE_PUNCH" {
				// Send acknowledgment
				ack := []byte("FYBRK_HOLE_PUNCH_ACK")
				conn.WriteToUDP(ack, clientAddr)
				fmt.Printf("Responded to hole punch from %s\n", clientAddr)
			}
		}
	}
}

// createSTUNRequest creates a STUN binding request packet
func (hp *HolePuncher) createSTUNRequest() []byte {
	// Simplified STUN binding request
	// In production, use a proper STUN library
	request := make([]byte, 20)
	
	// STUN header: Message Type (Binding Request) + Message Length + Magic Cookie + Transaction ID
	request[0] = 0x00 // Message Type: Binding Request (0x0001)
	request[1] = 0x01
	request[2] = 0x00 // Message Length: 0
	request[3] = 0x00
	
	// Magic Cookie (RFC 5389)
	request[4] = 0x21
	request[5] = 0x12
	request[6] = 0xA4
	request[7] = 0x42
	
	// Transaction ID (12 bytes, random)
	for i := 8; i < 20; i++ {
		request[i] = byte(i) // Simplified - should be random
	}
	
	return request
}

// parseSTUNResponse parses STUN response to extract public IP/port
func (hp *HolePuncher) parseSTUNResponse(data []byte) (*STUNResponse, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("invalid STUN response length")
	}
	
	// Simplified STUN response parsing
	// In production, use a proper STUN library
	
	// For now, return a mock response with the local address
	// This would be replaced with actual STUN parsing
	return &STUNResponse{
		PublicIP:   net.ParseIP("127.0.0.1"), // Mock - would be actual public IP
		PublicPort: hp.localAddr.Port,
	}, nil
}
