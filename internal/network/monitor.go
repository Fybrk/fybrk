package network

import (
	"context"
	"net"
	"sync"
	"time"
)

// ConnectionMonitor tracks connection quality and handles reconnection
type ConnectionMonitor struct {
	connections  map[string]*ConnectionInfo
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	onReconnect  func(deviceID string, conn net.Conn)
	onDisconnect func(deviceID string)
}

// ConnectionInfo tracks individual connection metrics
type ConnectionInfo struct {
	DeviceID    string
	Conn        net.Conn
	LastSeen    time.Time
	LastPing    time.Time
	PingLatency time.Duration
	PacketsSent int64
	PacketsRecv int64
	BytesSent   int64
	BytesRecv   int64
	Reconnects  int
	Quality     ConnectionQuality
	Address     string
}

// ConnectionQuality represents connection health
type ConnectionQuality int

const (
	QualityExcellent ConnectionQuality = iota
	QualityGood
	QualityPoor
	QualityDisconnected
)

// NewConnectionMonitor creates a new connection monitor
func NewConnectionMonitor() *ConnectionMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	cm := &ConnectionMonitor{
		connections: make(map[string]*ConnectionInfo),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start monitoring goroutine
	go cm.monitorLoop()

	return cm
}

// AddConnection adds a connection to monitor
func (cm *ConnectionMonitor) AddConnection(deviceID string, conn net.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[deviceID] = &ConnectionInfo{
		DeviceID: deviceID,
		Conn:     conn,
		LastSeen: time.Now(),
		LastPing: time.Now(),
		Quality:  QualityGood,
		Address:  conn.RemoteAddr().String(),
	}
}

// RemoveConnection removes a connection from monitoring
func (cm *ConnectionMonitor) RemoveConnection(deviceID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if info, exists := cm.connections[deviceID]; exists {
		info.Conn.Close()
		delete(cm.connections, deviceID)
	}
}

// UpdateActivity updates connection activity metrics
func (cm *ConnectionMonitor) UpdateActivity(deviceID string, bytesSent, bytesRecv int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if info, exists := cm.connections[deviceID]; exists {
		info.LastSeen = time.Now()
		info.BytesSent += bytesSent
		info.BytesRecv += bytesRecv
		info.PacketsSent++
		info.PacketsRecv++

		// Update quality based on activity
		cm.updateQuality(info)
	}
}

// SetReconnectHandler sets the callback for reconnection events
func (cm *ConnectionMonitor) SetReconnectHandler(handler func(deviceID string, conn net.Conn)) {
	cm.onReconnect = handler
}

// SetDisconnectHandler sets the callback for disconnection events
func (cm *ConnectionMonitor) SetDisconnectHandler(handler func(deviceID string)) {
	cm.onDisconnect = handler
}

// GetConnectionInfo returns connection information
func (cm *ConnectionMonitor) GetConnectionInfo(deviceID string) (*ConnectionInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	info, exists := cm.connections[deviceID]
	if exists {
		// Return a copy to avoid race conditions
		infoCopy := *info
		return &infoCopy, true
	}

	return nil, false
}

// GetAllConnections returns all connection information
func (cm *ConnectionMonitor) GetAllConnections() map[string]*ConnectionInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]*ConnectionInfo)
	for deviceID, info := range cm.connections {
		infoCopy := *info
		result[deviceID] = &infoCopy
	}

	return result
}

// monitorLoop runs the main monitoring loop
func (cm *ConnectionMonitor) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.checkConnections()
		}
	}
}

// checkConnections checks all connections for health and handles reconnection
func (cm *ConnectionMonitor) checkConnections() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()

	for deviceID, info := range cm.connections {
		// Check if connection is stale
		if now.Sub(info.LastSeen) > 30*time.Second {
			info.Quality = QualityPoor

			// Try to ping the connection
			if cm.pingConnection(info) {
				info.LastPing = now
				info.Quality = QualityGood
			} else {
				// Connection is dead, mark as disconnected
				info.Quality = QualityDisconnected

				// Attempt reconnection
				go cm.attemptReconnection(deviceID, info)
			}
		}

		// Update quality based on recent activity
		cm.updateQuality(info)
	}
}

// pingConnection sends a ping to test connection health
func (cm *ConnectionMonitor) pingConnection(info *ConnectionInfo) bool {
	if info.Conn == nil {
		return false
	}

	// Set a short deadline for ping
	info.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	info.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send ping message
	pingMsg := []byte("FYBRK_PING")
	start := time.Now()

	_, err := info.Conn.Write(pingMsg)
	if err != nil {
		return false
	}

	// Read pong response
	buffer := make([]byte, 64)
	n, err := info.Conn.Read(buffer)
	if err != nil {
		return false
	}

	if string(buffer[:n]) == "FYBRK_PONG" {
		info.PingLatency = time.Since(start)
		return true
	}

	return false
}

// updateQuality updates connection quality based on metrics
func (cm *ConnectionMonitor) updateQuality(info *ConnectionInfo) {
	now := time.Now()

	// Base quality on last activity and ping latency
	if now.Sub(info.LastSeen) < 10*time.Second {
		if info.PingLatency < 100*time.Millisecond {
			info.Quality = QualityExcellent
		} else if info.PingLatency < 500*time.Millisecond {
			info.Quality = QualityGood
		} else {
			info.Quality = QualityPoor
		}
	} else if now.Sub(info.LastSeen) < 30*time.Second {
		info.Quality = QualityPoor
	} else {
		info.Quality = QualityDisconnected
	}
}

// attemptReconnection tries to reconnect to a disconnected peer
func (cm *ConnectionMonitor) attemptReconnection(deviceID string, info *ConnectionInfo) {
	maxRetries := 5
	baseDelay := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Exponential backoff
		delay := baseDelay * time.Duration(1<<attempt)
		time.Sleep(delay)

		// Try to reconnect
		conn, err := net.DialTimeout("tcp", info.Address, 10*time.Second)
		if err != nil {
			continue
		}

		// Update connection info
		cm.mu.Lock()
		if existingInfo, exists := cm.connections[deviceID]; exists {
			existingInfo.Conn.Close() // Close old connection
			existingInfo.Conn = conn
			existingInfo.LastSeen = time.Now()
			existingInfo.Quality = QualityGood
			existingInfo.Reconnects++
		}
		cm.mu.Unlock()

		// Notify about successful reconnection
		if cm.onReconnect != nil {
			cm.onReconnect(deviceID, conn)
		}

		return
	}

	// Failed to reconnect, remove connection
	cm.mu.Lock()
	delete(cm.connections, deviceID)
	cm.mu.Unlock()

	// Notify about disconnection
	if cm.onDisconnect != nil {
		cm.onDisconnect(deviceID)
	}
}

// GetStats returns monitoring statistics
func (cm *ConnectionMonitor) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections": len(cm.connections),
		"quality_breakdown": map[string]int{
			"excellent":    0,
			"good":         0,
			"poor":         0,
			"disconnected": 0,
		},
		"total_bytes_sent": int64(0),
		"total_bytes_recv": int64(0),
		"total_reconnects": 0,
	}

	qualityBreakdown := stats["quality_breakdown"].(map[string]int)

	for _, info := range cm.connections {
		switch info.Quality {
		case QualityExcellent:
			qualityBreakdown["excellent"]++
		case QualityGood:
			qualityBreakdown["good"]++
		case QualityPoor:
			qualityBreakdown["poor"]++
		case QualityDisconnected:
			qualityBreakdown["disconnected"]++
		}

		stats["total_bytes_sent"] = stats["total_bytes_sent"].(int64) + info.BytesSent
		stats["total_bytes_recv"] = stats["total_bytes_recv"].(int64) + info.BytesRecv
		stats["total_reconnects"] = stats["total_reconnects"].(int) + info.Reconnects
	}

	return stats
}

// Close shuts down the connection monitor
func (cm *ConnectionMonitor) Close() error {
	cm.cancel()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Close all connections
	for _, info := range cm.connections {
		if info.Conn != nil {
			info.Conn.Close()
		}
	}

	cm.connections = make(map[string]*ConnectionInfo)
	return nil
}

// String returns a string representation of connection quality
func (q ConnectionQuality) String() string {
	switch q {
	case QualityExcellent:
		return "excellent"
	case QualityGood:
		return "good"
	case QualityPoor:
		return "poor"
	case QualityDisconnected:
		return "disconnected"
	default:
		return "unknown"
	}
}
