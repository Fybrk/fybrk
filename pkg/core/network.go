package core

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// NetworkManager handles P2P networking
type NetworkManager struct {
	fybrk      *Fybrk
	syncEngine *SyncEngine
	server     *http.Server
	upgrader   websocket.Upgrader
	port       int
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(fybrk *Fybrk, syncEngine *SyncEngine) *NetworkManager {
	return &NetworkManager{
		fybrk:      fybrk,
		syncEngine: syncEngine,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		port: 8080, // Default port
	}
}

// StartServer starts the WebSocket server for incoming connections
func (n *NetworkManager) StartServer() error {
	// Find available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	n.port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/sync", n.handleWebSocket)

	n.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", n.port),
		Handler: mux,
	}

	go func() {
		fmt.Printf("Server listening on port %d\n", n.port)
		if err := n.server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	return nil
}

// handleWebSocket handles incoming WebSocket connections
func (n *NetworkManager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := n.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade failed: %v\n", err)
		return
	}
	defer conn.Close()

	peerID := fmt.Sprintf("peer_%d", time.Now().UnixNano())
	peer := n.syncEngine.AddPeer(peerID)
	defer n.syncEngine.RemovePeer(peerID)

	// Start message sender goroutine
	go n.messageSender(conn, peer)

	// Handle incoming messages
	for {
		var msg SyncMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket error: %v\n", err)
			}
			break
		}

		if err := n.syncEngine.HandlePeerMessage(peerID, msg); err != nil {
			fmt.Printf("Error handling peer message: %v\n", err)
		}
	}
}

// messageSender sends messages to peer
func (n *NetworkManager) messageSender(conn *websocket.Conn, peer *Peer) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-peer.SendCh:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.WriteJSON(msg); err != nil {
				fmt.Printf("Error sending message: %v\n", err)
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ConnectToPeer connects to a remote peer
func (n *NetworkManager) ConnectToPeer(address string) error {
	url := fmt.Sprintf("ws://%s/sync", address)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}

	peerID := fmt.Sprintf("remote_%s", address)
	peer := n.syncEngine.AddPeer(peerID)

	go n.messageSender(conn, peer)
	go n.messageReceiver(conn, peerID)

	return nil
}

// messageReceiver receives messages from remote peer
func (n *NetworkManager) messageReceiver(conn *websocket.Conn, peerID string) {
	defer conn.Close()
	defer n.syncEngine.RemovePeer(peerID)

	for {
		var msg SyncMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket error: %v\n", err)
			}
			break
		}

		if err := n.syncEngine.HandlePeerMessage(peerID, msg); err != nil {
			fmt.Printf("Error handling peer message: %v\n", err)
		}
	}
}

// GetLocalAddress returns the local server address
func (n *NetworkManager) GetLocalAddress() string {
	return fmt.Sprintf("localhost:%d", n.port)
}

// Stop stops the network manager
func (n *NetworkManager) Stop() error {
	if n.server != nil {
		return n.server.Close()
	}
	return nil
}
