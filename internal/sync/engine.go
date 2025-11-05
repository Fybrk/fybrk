package sync

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Fybrk/fybrk/internal/storage"
	"github.com/Fybrk/fybrk/internal/watcher"
	"github.com/Fybrk/fybrk/pkg/types"
)

type Engine struct {
	metadataStore *storage.MetadataStore
	chunker       *storage.Chunker
	encryptor     *storage.Encryptor
	watcher       *watcher.FileWatcher
	syncPath      string
	deviceID      string
	multiDevice   *MultiDeviceSync
}

func NewEngine(metadataStore *storage.MetadataStore, chunker *storage.Chunker, encryptor *storage.Encryptor, syncPath, deviceID string) (*Engine, error) {
	fileWatcher, err := watcher.NewFileWatcher()
	if err != nil {
		return nil, err
	}

	engine := &Engine{
		metadataStore: metadataStore,
		chunker:       chunker,
		encryptor:     encryptor,
		watcher:       fileWatcher,
		syncPath:      syncPath,
		deviceID:      deviceID,
	}

	// Start watching the sync path
	if err := fileWatcher.AddPath(syncPath); err != nil {
		return nil, err
	}

	go engine.handleFileEvents()
	return engine, nil
}

func (e *Engine) EnableMultiDeviceSync(port int) error {
	mds, err := NewMultiDeviceSync(e, e.encryptor, e.deviceID, port)
	if err != nil {
		return err
	}
	
	e.multiDevice = mds
	return mds.Start()
}

func (e *Engine) GetConnectedDevices() []string {
	if e.multiDevice == nil {
		return []string{}
	}
	return e.multiDevice.GetConnectedDevices()
}

func (e *Engine) handleFileEvents() {
	for {
		select {
		case event := <-e.watcher.Events():
			e.processFileEvent(event)
		case err := <-e.watcher.Errors():
			fmt.Printf("File watcher error: %v\n", err)
		}
	}
}

func (e *Engine) processFileEvent(event watcher.FileEvent) {
	// Skip if file is outside sync path
	relPath, err := filepath.Rel(e.syncPath, event.Path)
	if err != nil || filepath.IsAbs(relPath) {
		return
	}

	switch event.Operation {
	case "create", "write":
		e.handleFileChange(event.Path)
	case "remove":
		e.handleFileRemoval(event.Path)
	}
}

func (e *Engine) handleFileChange(filePath string) {
	// Skip directories
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return
	}

	// Get relative path for storage
	relPath, err := filepath.Rel(e.syncPath, filePath)
	if err != nil {
		return
	}

	// Check if file has changed
	existingMetadata, err := e.metadataStore.GetFileMetadata(relPath)
	if err == nil {
		// File exists in metadata, check if it's actually changed
		if info.ModTime().Equal(existingMetadata.ModTime) && info.Size() == existingMetadata.Size {
			return // No change
		}
	}

	// Process the file
	if err := e.processFile(filePath, relPath); err != nil {
		fmt.Printf("Error processing file %s: %v\n", filePath, err)
	}
}

func (e *Engine) handleFileRemoval(filePath string) {
	relPath, err := filepath.Rel(e.syncPath, filePath)
	if err != nil {
		return
	}

	// Remove from metadata store
	if err := e.metadataStore.DeleteFileMetadata(relPath); err != nil {
		fmt.Printf("Error removing file metadata %s: %v\n", relPath, err)
	}
}

func (e *Engine) processFile(filePath, relPath string) error {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Chunk the file
	chunks, err := e.chunker.ChunkFile(filePath)
	if err != nil {
		return err
	}

	// Encrypt chunks
	for i := range chunks {
		if err := e.encryptor.EncryptChunk(&chunks[i]); err != nil {
			return err
		}
	}

	// Calculate file hash
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	fileHash := sha256.Sum256(fileData)

	// Create chunk hashes
	chunkHashes := make([][32]byte, len(chunks))
	for i, chunk := range chunks {
		chunkHashes[i] = chunk.Hash
	}

	// Get version number
	version := int64(1)
	if existingMetadata, err := e.metadataStore.GetFileMetadata(relPath); err == nil {
		version = existingMetadata.Version + 1
	}

	// Create metadata
	metadata := &types.FileMetadata{
		Path:    relPath,
		Hash:    fileHash,
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Chunks:  chunkHashes,
		Version: version,
	}

	// Store metadata
	return e.metadataStore.StoreFileMetadata(metadata)
}

func (e *Engine) ScanDirectory() error {
	return filepath.Walk(e.syncPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(e.syncPath, path)
		if err != nil {
			return err
		}

		return e.processFile(path, relPath)
	})
}

func (e *Engine) GetSyncedFiles() ([]*types.FileMetadata, error) {
	return e.metadataStore.ListFiles()
}

func (e *Engine) Close() error {
	if e.multiDevice != nil {
		e.multiDevice.Stop()
	}
	return e.watcher.Close()
}
