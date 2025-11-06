package core

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SyncEngine handles bidirectional file synchronization
type SyncEngine struct {
	fybrk   *Fybrk
	watcher *Watcher
	peers   map[string]*Peer
	running bool
}

// Peer represents a connected peer
type Peer struct {
	ID       string
	LastSeen time.Time
	SendCh   chan SyncMessage
}

// SyncMessage represents a sync message between peers
type SyncMessage struct {
	Type     MessageType `json:"type"`
	Path     string      `json:"path"`
	Hash     string      `json:"hash"`
	Size     int64       `json:"size"`
	ModTime  time.Time   `json:"mod_time"`
	Content  []byte      `json:"content,omitempty"`
	Checksum string      `json:"checksum"`
}

// MessageType represents the type of sync message
type MessageType string

const (
	MsgFileCreate MessageType = "file_create"
	MsgFileModify MessageType = "file_modify"
	MsgFileDelete MessageType = "file_delete"
	MsgFileList   MessageType = "file_list"
	MsgFileReq    MessageType = "file_request"
)

// NewSyncEngine creates a new sync engine
func NewSyncEngine(fybrk *Fybrk) (*SyncEngine, error) {
	watcher, err := NewWatcher(fybrk)
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &SyncEngine{
		fybrk:   fybrk,
		watcher: watcher,
		peers:   make(map[string]*Peer),
	}, nil
}

// Start begins the sync engine
func (s *SyncEngine) Start() error {
	if s.running {
		return fmt.Errorf("sync engine already running")
	}

	// Perform initial scan
	fmt.Println("Scanning files...")
	if err := s.watcher.InitialScan(); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	s.running = true
	go s.syncLoop()

	fmt.Println("Sync engine started")
	return nil
}

// syncLoop is the main sync processing loop
func (s *SyncEngine) syncLoop() {
	for s.running {
		select {
		case event := <-s.watcher.Events():
			s.handleFileEvent(event)
		case <-time.After(1 * time.Second):
			// Periodic maintenance
		}
	}
}

// handleFileEvent processes file system events
func (s *SyncEngine) handleFileEvent(event FileEvent) {
	fmt.Printf("File event: %s %s\n", event.Type, event.Path)

	// Update local database
	switch event.Type {
	case EventCreate, EventModify:
		if err := s.fybrk.updateFileRecord(event.Path, event.Size, event.ModTime, event.Hash); err != nil {
			fmt.Printf("Error updating file record: %v\n", err)
			return
		}
	case EventDelete:
		if err := s.fybrk.deleteFileRecord(event.Path); err != nil {
			fmt.Printf("Error deleting file record: %v\n", err)
			return
		}
	}

	// Broadcast to peers
	s.broadcastEvent(event)
}

// broadcastEvent sends file event to all connected peers
func (s *SyncEngine) broadcastEvent(event FileEvent) {
	var msgType MessageType
	switch event.Type {
	case EventCreate:
		msgType = MsgFileCreate
	case EventModify:
		msgType = MsgFileModify
	case EventDelete:
		msgType = MsgFileDelete
	default:
		return
	}

	msg := SyncMessage{
		Type:    msgType,
		Path:    event.Path,
		Hash:    event.Hash,
		Size:    event.Size,
		ModTime: event.ModTime,
	}

	// For create/modify, include file content for small files
	if event.Type != EventDelete && event.Size < 1024*1024 { // 1MB limit
		content, err := s.readFileContent(event.Path)
		if err == nil {
			msg.Content = content
		}
	}

	// Send to all peers
	for _, peer := range s.peers {
		select {
		case peer.SendCh <- msg:
		default:
			// Peer channel full, skip
		}
	}
}

// readFileContent reads file content
func (s *SyncEngine) readFileContent(relPath string) ([]byte, error) {
	fullPath := filepath.Join(s.fybrk.syncPath, relPath)
	return os.ReadFile(fullPath)
}

// HandlePeerMessage processes incoming messages from peers
func (s *SyncEngine) HandlePeerMessage(peerID string, msg SyncMessage) error {
	fmt.Printf("Peer message from %s: %s %s\n", peerID, msg.Type, msg.Path)

	switch msg.Type {
	case MsgFileCreate, MsgFileModify:
		return s.handleFileUpdate(msg)
	case MsgFileDelete:
		return s.handleFileDelete(msg)
	case MsgFileReq:
		return s.handleFileRequest(peerID, msg)
	}

	return nil
}

// handleFileUpdate processes file create/modify from peer
func (s *SyncEngine) handleFileUpdate(msg SyncMessage) error {
	fullPath := filepath.Join(s.fybrk.syncPath, msg.Path)

	// Check if we need this update
	existing, err := s.fybrk.getFileRecord(msg.Path)
	if err == nil && existing.Hash == msg.Hash {
		return nil // Already have this version
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file content
	if len(msg.Content) > 0 {
		if err := os.WriteFile(fullPath, msg.Content, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		// Set modification time
		if err := os.Chtimes(fullPath, msg.ModTime, msg.ModTime); err != nil {
			fmt.Printf("Warning: failed to set file times: %v\n", err)
		}

		// Update database
		return s.fybrk.updateFileRecord(msg.Path, msg.Size, msg.ModTime, msg.Hash)
	}

	return nil
}

// handleFileDelete processes file delete from peer
func (s *SyncEngine) handleFileDelete(msg SyncMessage) error {
	fullPath := filepath.Join(s.fybrk.syncPath, msg.Path)

	// Remove file
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Update database
	return s.fybrk.deleteFileRecord(msg.Path)
}

// handleFileRequest processes file request from peer
func (s *SyncEngine) handleFileRequest(peerID string, msg SyncMessage) error {
	peer, exists := s.peers[peerID]
	if !exists {
		return fmt.Errorf("unknown peer: %s", peerID)
	}

	content, err := s.readFileContent(msg.Path)
	if err != nil {
		return err
	}

	response := SyncMessage{
		Type:    MsgFileModify,
		Path:    msg.Path,
		Content: content,
	}

	select {
	case peer.SendCh <- response:
	default:
		return fmt.Errorf("peer channel full")
	}

	return nil
}

// AddPeer adds a new peer connection
func (s *SyncEngine) AddPeer(peerID string) *Peer {
	peer := &Peer{
		ID:       peerID,
		LastSeen: time.Now(),
		SendCh:   make(chan SyncMessage, 100),
	}
	s.peers[peerID] = peer

	fmt.Printf("Peer connected: %s\n", peerID)
	return peer
}

// RemovePeer removes a peer connection
func (s *SyncEngine) RemovePeer(peerID string) {
	if peer, exists := s.peers[peerID]; exists {
		close(peer.SendCh)
		delete(s.peers, peerID)
		fmt.Printf("Peer disconnected: %s\n", peerID)
	}
}

// Stop stops the sync engine
func (s *SyncEngine) Stop() error {
	s.running = false
	return s.watcher.Close()
}
