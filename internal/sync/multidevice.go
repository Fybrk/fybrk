package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Fybrk/fybrk/internal/network"
	"github.com/Fybrk/fybrk/internal/storage"
	"github.com/Fybrk/fybrk/pkg/types"
)

type MultiDeviceSync struct {
	engine    *Engine
	network   *network.PeerNetwork
	encryptor *storage.Encryptor
	deviceID  string
}

type SyncMessage struct {
	Type     string                `json:"type"`
	Files    []*types.FileMetadata `json:"files,omitempty"`
	Request  *FileRequest          `json:"request,omitempty"`
	Response *FileResponse         `json:"response,omitempty"`
}

type FileRequest struct {
	Path   string     `json:"path"`
	Chunks [][32]byte `json:"chunks"`
}

type FileResponse struct {
	Path   string        `json:"path"`
	Chunks []types.Chunk `json:"chunks"`
}

func NewMultiDeviceSync(engine *Engine, encryptor *storage.Encryptor, deviceID string, port int) (*MultiDeviceSync, error) {
	peerNetwork := network.NewPeerNetwork(deviceID, port)
	
	mds := &MultiDeviceSync{
		engine:    engine,
		network:   peerNetwork,
		encryptor: encryptor,
		deviceID:  deviceID,
	}
	
	peerNetwork.SetMessageHandler(mds.handleMessage)
	
	return mds, nil
}

func (mds *MultiDeviceSync) Start() error {
	if err := mds.network.Start(); err != nil {
		return err
	}
	
	// Start periodic sync
	go mds.periodicSync()
	
	return nil
}

func (mds *MultiDeviceSync) Stop() error {
	return mds.network.Stop()
}

func (mds *MultiDeviceSync) periodicSync() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		mds.broadcastFileList()
	}
}

func (mds *MultiDeviceSync) broadcastFileList() {
	files, err := mds.engine.GetSyncedFiles()
	if err != nil {
		log.Printf("Error getting synced files: %v", err)
		return
	}
	
	syncMsg := SyncMessage{
		Type:  "file_list",
		Files: files,
	}
	
	msg := &network.Message{
		Type:      "sync",
		DeviceID:  mds.deviceID,
		Timestamp: time.Now(),
		Data:      syncMsg,
	}
	
	mds.network.BroadcastMessage(msg)
}

func (mds *MultiDeviceSync) handleMessage(deviceID string, msg *network.Message) {
	if msg.Type != "sync" {
		return
	}
	
	// Parse sync message
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		log.Printf("Error marshaling sync data: %v", err)
		return
	}
	
	var syncMsg SyncMessage
	if err := json.Unmarshal(dataBytes, &syncMsg); err != nil {
		log.Printf("Error unmarshaling sync message: %v", err)
		return
	}
	
	switch syncMsg.Type {
	case "file_list":
		mds.handleFileList(deviceID, syncMsg.Files)
	case "file_request":
		mds.handleFileRequest(deviceID, syncMsg.Request)
	case "file_response":
		mds.handleFileResponse(deviceID, syncMsg.Response)
	}
}

func (mds *MultiDeviceSync) handleFileList(deviceID string, remoteFiles []*types.FileMetadata) {
	localFiles, err := mds.engine.GetSyncedFiles()
	if err != nil {
		log.Printf("Error getting local files: %v", err)
		return
	}
	
	// Create map of local files for quick lookup
	localFileMap := make(map[string]*types.FileMetadata)
	for _, file := range localFiles {
		localFileMap[file.Path] = file
	}
	
	// Check for files we need
	for _, remoteFile := range remoteFiles {
		localFile, exists := localFileMap[remoteFile.Path]
		
		if !exists || localFile.Version < remoteFile.Version {
			// Request this file
			mds.requestFile(deviceID, remoteFile)
		}
	}
}

func (mds *MultiDeviceSync) requestFile(deviceID string, fileMetadata *types.FileMetadata) {
	request := &FileRequest{
		Path:   fileMetadata.Path,
		Chunks: fileMetadata.Chunks,
	}
	
	syncMsg := SyncMessage{
		Type:    "file_request",
		Request: request,
	}
	
	msg := &network.Message{
		Type:      "sync",
		DeviceID:  mds.deviceID,
		Timestamp: time.Now(),
		Data:      syncMsg,
	}
	
	if err := mds.network.SendMessage(deviceID, msg); err != nil {
		log.Printf("Error sending file request: %v", err)
	}
}

func (mds *MultiDeviceSync) handleFileRequest(deviceID string, request *FileRequest) {
	// Get chunks for requested file
	chunks, err := mds.getFileChunks(request.Path, request.Chunks)
	if err != nil {
		log.Printf("Error getting file chunks: %v", err)
		return
	}
	
	response := &FileResponse{
		Path:   request.Path,
		Chunks: chunks,
	}
	
	syncMsg := SyncMessage{
		Type:     "file_response",
		Response: response,
	}
	
	msg := &network.Message{
		Type:      "sync",
		DeviceID:  mds.deviceID,
		Timestamp: time.Now(),
		Data:      syncMsg,
	}
	
	if err := mds.network.SendMessage(deviceID, msg); err != nil {
		log.Printf("Error sending file response: %v", err)
	}
}

func (mds *MultiDeviceSync) handleFileResponse(deviceID string, response *FileResponse) {
	// Decrypt chunks
	for i := range response.Chunks {
		if err := mds.encryptor.DecryptChunk(&response.Chunks[i]); err != nil {
			log.Printf("Error decrypting chunk: %v", err)
			return
		}
	}
	
	// Reassemble file
	chunker := storage.NewChunker(storage.DefaultChunkSize)
	fileData, err := chunker.ReassembleChunks(response.Chunks)
	if err != nil {
		log.Printf("Error reassembling file: %v", err)
		return
	}
	
	// Write file to sync directory
	if err := mds.writeReceivedFile(response.Path, fileData); err != nil {
		log.Printf("Error writing received file: %v", err)
		return
	}
	
	log.Printf("Successfully synced file: %s", response.Path)
}

func (mds *MultiDeviceSync) getFileChunks(path string, chunkHashes [][32]byte) ([]types.Chunk, error) {
	// Get file metadata from storage
	fileMetadata, err := mds.engine.metadataStore.GetFileMetadata(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %v", err)
	}
	
	// Create chunker to read chunks
	chunker := storage.NewChunker(storage.DefaultChunkSize)
	
	// Read and chunk the file
	chunks, err := chunker.ChunkFile(fileMetadata.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk file: %v", err)
	}
	
	// Encrypt chunks
	for i := range chunks {
		if err := mds.encryptor.EncryptChunk(&chunks[i]); err != nil {
			return nil, fmt.Errorf("failed to encrypt chunk: %v", err)
		}
	}
	
	return chunks, nil
}

func (mds *MultiDeviceSync) writeReceivedFile(path string, data []byte) error {
	// Get the sync directory from the engine
	syncDir := mds.engine.syncPath
	
	// Create full path within sync directory
	fullPath := filepath.Join(syncDir, filepath.Base(path))
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	
	// Write file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	
	// Update metadata store
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}
	
	// Create chunker to get chunk hashes
	chunker := storage.NewChunker(storage.DefaultChunkSize)
	chunks, err := chunker.ChunkFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to chunk file: %v", err)
	}
	
	// Create chunk hashes
	chunkHashes := make([][32]byte, len(chunks))
	for i, chunk := range chunks {
		chunkHashes[i] = chunk.Hash
	}
	
	// Store metadata
	metadata := &types.FileMetadata{
		Path:    fullPath,
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
		Chunks:  chunkHashes,
		Version: 1,
	}
	
	return mds.engine.metadataStore.StoreFileMetadata(metadata)
}

func (mds *MultiDeviceSync) GetConnectedDevices() []string {
	return mds.network.GetPeers()
}
