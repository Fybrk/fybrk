# Fybrk

Secure, distributed file synchronization tool.

## Features

- AES-256-GCM encryption
- SHA-256 file integrity
- SQLite metadata storage
- Real-time file monitoring
- Version tracking
- Cross-platform support

## Installation

Download the latest release from GitHub or build from source:

```bash
go build -o fybrk ./cli/cmd/fybrk
```

## Usage

**Initialize a sync directory:**
```bash
fybrk -path /path/to/sync -cmd scan
```

**Start synchronization:**
```bash
fybrk -path /path/to/sync
```

**List synchronized files:**
```bash
fybrk -path /path/to/sync -cmd list
```

## Architecture

```
fybrk/
├── cli/            # Command-line interface
├── internal/       # Core synchronization engine
│   ├── storage/    # Encryption, chunking, metadata
│   ├── sync/       # Synchronization logic
│   └── watcher/    # File system monitoring
└── pkg/           # Public API
    ├── fybrk/     # Main API
    └── types/     # Data structures
```

## Development

**Requirements:**
- Go 1.21+
- SQLite3

**Commands:**
```bash
make build         # Build binary
make test          # Run tests
make clean         # Clean build artifacts
```

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

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT
