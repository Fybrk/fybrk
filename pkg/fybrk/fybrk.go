package fybrk

import (
	"github.com/Fybrk/fybrk/internal/storage"
	"github.com/Fybrk/fybrk/internal/sync"
	"github.com/Fybrk/fybrk/pkg/types"
)

// Client provides the main API for Fybrk operations
type Client struct {
	metadataStore *storage.MetadataStore
	chunker       *storage.Chunker
	encryptor     *storage.Encryptor
	engine        *sync.Engine
}

// Config holds configuration for Fybrk client
type Config struct {
	SyncPath  string
	DBPath    string
	DeviceID  string
	ChunkSize int
	Key       []byte
}

// NewClient creates a new Fybrk client
func NewClient(config *Config) (*Client, error) {
	// Create metadata store
	metadataStore, err := storage.NewMetadataStore(config.DBPath)
	if err != nil {
		return nil, err
	}

	// Create chunker
	chunker := storage.NewChunker(config.ChunkSize)

	// Create encryptor
	encryptor, err := storage.NewEncryptor(config.Key)
	if err != nil {
		return nil, err
	}

	// Create sync engine
	engine, err := sync.NewEngine(metadataStore, chunker, encryptor, config.SyncPath, config.DeviceID)
	if err != nil {
		return nil, err
	}

	return &Client{
		metadataStore: metadataStore,
		chunker:       chunker,
		encryptor:     encryptor,
		engine:        engine,
	}, nil
}

// ScanDirectory scans the sync directory for changes
func (c *Client) ScanDirectory() error {
	return c.engine.ScanDirectory()
}

// GetSyncedFiles returns all synced files
func (c *Client) GetSyncedFiles() ([]*types.FileMetadata, error) {
	return c.engine.GetSyncedFiles()
}

// EnableMultiDeviceSync enables peer-to-peer synchronization
func (c *Client) EnableMultiDeviceSync(port int) error {
	return c.engine.EnableMultiDeviceSync(port)
}

// GetConnectedDevices returns list of connected device IDs
func (c *Client) GetConnectedDevices() []string {
	return c.engine.GetConnectedDevices()
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	if err := c.engine.Close(); err != nil {
		return err
	}
	return c.metadataStore.Close()
}
