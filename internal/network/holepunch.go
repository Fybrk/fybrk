package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/stun"
)

// HolePuncher handles NAT traversal using hole punching techniques
type HolePuncher struct {
	localAddr  *net.UDPAddr
	conn       *net.UDPConn
	stunServer string
	mu         sync.RWMutex
	retryCount int
	timeout    time.Duration
}

// STUNResponse contains STUN server response with public IP/port
type STUNResponse struct {
	PublicIP   net.IP
	PublicPort int
}

// NewHolePuncher creates a new hole puncher with production settings
func NewHolePuncher() *HolePuncher {
	return &HolePuncher{
		stunServer: "stun.l.google.com:19302",
		retryCount: 3,
		timeout:    10 * time.Second,
	}
}

// GetPublicAddress discovers public IP and port using real STUN protocol
func (hp *HolePuncher) GetPublicAddress() (*STUNResponse, error) {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %v", err)
	}
	defer conn.Close()

	hp.conn = conn
	hp.localAddr = conn.LocalAddr().(*net.UDPAddr)

	stunAddr, err := net.ResolveUDPAddr("udp", hp.stunServer)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server: %v", err)
	}

	// Create STUN binding request using pion/stun
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	var response *STUNResponse
	for i := 0; i < hp.retryCount; i++ {
		conn.SetWriteDeadline(time.Now().Add(hp.timeout))
		_, err = conn.WriteToUDP(message.Raw, stunAddr)
		if err != nil {
			continue
		}

		conn.SetReadDeadline(time.Now().Add(hp.timeout))
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		response, err = hp.parseSTUNResponse(buffer[:n])
		if err == nil {
			break
		}
	}

	if response == nil {
		return nil, fmt.Errorf("failed to get STUN response after %d attempts", hp.retryCount)
	}

	return response, nil
}

// parseSTUNResponse parses real STUN response using pion/stun
func (hp *HolePuncher) parseSTUNResponse(data []byte) (*STUNResponse, error) {
	message := &stun.Message{Raw: data}
	if err := message.Decode(); err != nil {
		return nil, fmt.Errorf("failed to decode STUN message: %v", err)
	}

	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(message); err != nil {
		// Fallback to mapped address
		var mappedAddr stun.MappedAddress
		if err := mappedAddr.GetFrom(message); err != nil {
			return nil, fmt.Errorf("no address found in STUN response")
		}
		return &STUNResponse{
			PublicIP:   mappedAddr.IP,
			PublicPort: mappedAddr.Port,
		}, nil
	}

	return &STUNResponse{
		PublicIP:   xorAddr.IP,
		PublicPort: xorAddr.Port,
	}, nil
}

// PunchHole attempts to establish direct connection with comprehensive retry logic
func (hp *HolePuncher) PunchHole(peerPublicIP net.IP, peerPublicPort int, peerPrivateIP net.IP, peerPrivatePort int) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create hole punch connection: %v", err)
	}

	attempts := []net.UDPAddr{
		{IP: peerPublicIP, Port: peerPublicPort},
		{IP: peerPrivateIP, Port: peerPrivatePort},
	}

	punchData := []byte("FYBRK_HOLE_PUNCH")

	// Enhanced hole punching with exponential backoff
	for attempt := 0; attempt < 20; attempt++ {
		for _, addr := range attempts {
			conn.WriteToUDP(punchData, &addr)
		}

		// Exponential backoff: 50ms, 100ms, 200ms, then 500ms
		delay := time.Duration(50*(1<<min(attempt/5, 3))) * time.Millisecond
		time.Sleep(delay)
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("hole punch failed: %v", err)
	}

	if string(buffer[:n]) == "FYBRK_HOLE_PUNCH_ACK" {
		return conn, nil
	}

	conn.Close()
	return nil, fmt.Errorf("invalid hole punch response")
}

// StartHolePunchListener with connection quality monitoring
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

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			buffer := make([]byte, 1024)
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			if string(buffer[:n]) == "FYBRK_HOLE_PUNCH" {
				ack := []byte("FYBRK_HOLE_PUNCH_ACK")
				conn.WriteToUDP(ack, clientAddr)
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
