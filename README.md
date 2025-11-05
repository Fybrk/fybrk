# Fybrk Core

Synchronization engine for distributed file storage.

## Features

- AES-256-GCM encryption
- SHA-256 file integrity
- SQLite metadata storage
- Real-time file monitoring
- Version tracking
- Cross-platform support

## Installation

```bash
go get github.com/Fybrk/core
```

## Usage

```go
import "github.com/Fybrk/core/pkg/fybrk"

config := &fybrk.Config{
    SyncPath:  "/path/to/sync",
    DBPath:    "/path/to/metadata.db",
    DeviceID:  "device-identifier",
    ChunkSize: 1024 * 1024,
    Key:       encryptionKey,
}

client, err := fybrk.NewClient(config)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Scan for changes
err = client.ScanDirectory()
if err != nil {
    log.Fatal(err)
}

// List synchronized files
files, err := client.GetSyncedFiles()
if err != nil {
    log.Fatal(err)
}
```

## Architecture

```
core/
├── internal/
│   ├── storage/    # Encryption, chunking, metadata
│   ├── sync/       # Synchronization logic
│   └── watcher/    # File system monitoring
└── pkg/
    ├── fybrk/      # Public API
    └── types/      # Data structures
```

## Development

**Requirements:**
- Go 1.21+
- SQLite3

**Commands:**
```bash
make test           # Run tests
make test-coverage  # Generate coverage report
make lint          # Code analysis
```

## Testing

The library includes comprehensive tests covering:
- File chunking and encryption
- Metadata operations
- File system monitoring
- Synchronization logic

Target coverage: 80%+

## Configuration

Fybrk stores metadata in a `.fybrk` directory:

```
/sync/path/
├── .fybrk/
│   ├── metadata.db    # File metadata
│   └── key           # Encryption key
└── [your files]
```

## Security

- Files encrypted before storage
- Keys generated locally
- SHA-256 integrity verification
- No plaintext transmission

## API Reference

### Client

- `NewClient(config)` - Initialize client
- `ScanDirectory()` - Detect file changes
- `GetSyncedFiles()` - List tracked files
- `Close()` - Release resources

### Configuration

- `SyncPath` - Directory to synchronize
- `DBPath` - Metadata database location
- `DeviceID` - Unique device identifier
- `ChunkSize` - File chunk size (bytes)
- `Key` - 32-byte encryption key

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT
