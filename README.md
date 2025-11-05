# Fybrk - Secure Peer-to-Peer File Synchronization

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![Test Coverage](https://img.shields.io/badge/coverage-59.1%25-yellow.svg)](https://github.com/Fybrk/fybrk)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/Fybrk/fybrk)

**State-of-the-art P2P file sync with internet-wide connectivity and zero-trust security.**

## ✨ Features

### 🌐 Internet-Wide Connectivity

- **Global Sync**: Works across different networks, not just local
- **QR Code Pairing**: Instant device pairing with beautiful terminal QR codes
- **NAT Traversal**: Automatic hole punching through firewalls using STUN
- **Bootstrap Network**: Decentralized discovery with multiple fallback methods
- **DHT Integration**: BitTorrent DHT for fully decentralized operation

### 🔒 Production Security

- **End-to-End Encryption**: AES-256-GCM with SHA-256 integrity verification
- **Zero-Trust Architecture**: Bootstrap servers never see file data
- **Temporary Rendezvous**: QR codes expire in 10 minutes for security
- **Perfect Forward Secrecy**: Unique keys per session
- **Local Key Storage**: Encryption keys never leave devices

### 🚀 Enterprise Features

- **Auto-Reconnection**: Intelligent connection monitoring and recovery
- **Connection Quality**: Real-time health checks with ping/pong
- **Performance Monitoring**: Bandwidth, latency, and connection tracking
- **Error Handling**: Comprehensive retry logic with exponential backoff
- **Service Redundancy**: Multiple bootstrap nodes with failover

### 🔧 Developer Ready

- **Comprehensive API**: Clean Go library interface
- **Extensive Testing**: 59.1% overall coverage, 89%+ for critical components
- **CLI Tool**: Full command-line interface for automation
- **Cross-Platform**: Works on Windows, macOS, and Linux

## 🚀 Quick Start

### Installation

**Auto-detect from Pre-built Binaries:**

```bash
curl -sSL https://fybrk.com/install.sh | bash
```

**From Source:**

```bash
git clone https://github.com/Fybrk/fybrk.git
cd fybrk
make build
```

### Basic Usage

**Device A (has files):**

```bash
fybrk init    # Initialize current directory
fybrk pair    # Generate QR code for pairing
fybrk sync    # Start real-time sync
```

**Device B (wants to sync):**

```bash
fybrk pair-with '<QR-CODE-DATA>'  # Join from QR code (works over internet!)
fybrk sync                        # Start syncing
```

## 📋 Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize directory for sync (first-time setup) |
| `sync` | Start real-time synchronization (default) |
| `pair` | Generate QR code to pair with other devices |
| `pair-with` | Join sync network from QR code |
| `list` | List all tracked files and their status |

## 🏗️ Architecture

### Internet-Wide Connectivity

```
┌─────────────────┐    ┌─────────────────┐
│    Device A     │    │    Device B     │
│                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Fybrk     │◄┼────┼►│   Fybrk     │ │
│ │   Client    │ │    │ │   Client    │ │
│ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘
         │                       │
         └───────────┬───────────┘
                     │
         ┌───────────────────┐
         │ Bootstrap Network │
         │ (Discovery Only)  │
         │                   │
         │ • STUN Servers    │
         │ • DHT Network     │
         │ • Rendezvous      │
         └───────────────────┘
```

### Core Components

- **File Watcher**: Real-time change detection with fsnotify
- **Storage Engine**: Chunking, encryption, and metadata management
- **Network Layer**: P2P discovery and communication
- **Sync Engine**: Multi-device file synchronization
- **Bootstrap Service**: Internet-wide device discovery
- **Connection Monitor**: Quality tracking and auto-reconnection

## 🔒 Security Model

- **AES-256 Encryption**: Military-grade file encryption
- **Client-Side Keys**: 32-byte keys generated and stored locally
- **No Data Leakage**: Bootstrap servers only handle discovery
- **Expiring Tokens**: Time-limited pairing for security
- **Integrity Verification**: SHA-256 checksums prevent corruption

## 📊 Production Features

### Error Handling

- Comprehensive retry logic with exponential backoff
- Multiple bootstrap node fallbacks
- Graceful degradation when services fail
- Detailed error reporting and logging

### Performance Monitoring

- Real-time connection quality metrics (Excellent/Good/Poor/Disconnected)
- Bandwidth and latency tracking
- Service health statistics
- Connection success/failure rates

### Reliability

- Automatic reconnection on network changes
- Connection health monitoring with ping/pong
- Service redundancy and failover
- Robust NAT traversal with hole punching

## 🛠️ Development

### Building

```bash
make build          # Build binary
make test           # Run tests
make coverage       # Generate coverage report
```

### Testing

```bash
# Integration tests
go test ./...

# Coverage analysis (59.1% overall, 89%+ critical)
make coverage
```

## 📈 Roadmap

- [x] Internet-wide sync with QR pairing
- [x] Real STUN protocol implementation
- [x] Connection quality monitoring
- [x] Production-grade error handling
- [ ] Full DHT integration
- [ ] Mobile app integration
- [ ] Web interface
- [ ] Plugin system

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🔗 Links

- **Documentation**: [fybrk.com](https://fybrk.com)

---

**Fybrk** - Secure, fast, and reliable peer-to-peer file synchronization.

### Using Go

```bash
go install github.com/Fybrk/fybrk/cli/cmd/fybrk@latest
```

## Quick Start

### 1. First Time Setup - Initialize Directory

```bash
fybrk /your/sync/folder scan
```

This creates a `.fybrk` folder with encryption keys and scans all files.

### 2. Start Synchronization

```bash
fybrk /your/sync/folder sync
# or simply (sync is the default)
fybrk /your/sync/folder
```

This starts real-time monitoring and syncing with other devices.

### 3. Check What's Being Synced

```bash
fybrk /your/sync/folder list
```

Shows all tracked files with version and chunk information.

### 4. Multi-Device Setup

1. **First device**: Run `fybrk /your/folder scan` to initialize
2. **Copy the folder** to other devices (including the `.fybrk` folder)
3. **Each device**: Run `fybrk /your/folder sync`
4. Devices automatically discover each other and sync changes

### What Each Command Does

- **scan**: First-time setup - creates `.fybrk` folder, generates encryption key, scans all files
- **sync**: Starts real-time file monitoring and peer-to-peer synchronization  
- **list**: Shows all files being tracked with their version and status

### Alternative Command Format

You can also use command-first format:

```bash
fybrk scan /your/sync/folder    # Initialize directory
fybrk sync /your/sync/folder    # Start syncing
fybrk list /your/sync/folder    # List tracked files
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Device A      │    │   Device B      │    │   Device C      │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Fybrk Core  │ │◄──►│ │ Fybrk Core  │ │◄──►│ │ Fybrk Core  │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Sync Folder │ │    │ │ Sync Folder │ │    │ │ Sync Folder │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Core Components

- **Sync Engine**: File monitoring, change detection, conflict resolution
- **Storage Layer**: Encryption, chunking, metadata management with SQLite
- **Network Layer**: P2P discovery, UPnP NAT traversal, secure messaging
- **Pairing System**: QR code-based device authentication
- **CLI Interface**: Command-line tool for all operations

## Test Coverage

| Component | Coverage | Status |
|-----------|----------|---------|
| **Storage** | 89.4% | Excellent |
| **Watcher** | 89.3% | Excellent |
| **Pairing** | 88.1% | Excellent |
| **Public API** | 83.3% | Good |
| **Sync Engine** | 62.6% | Good |
| **Network** | 60.1% | Acceptable |
| **Overall** | 59.1% | Acceptable |

## Development

### Prerequisites

- Go 1.21 or later
- Make (for build automation)

### Building

```bash
make build          # Build binaries
make test           # Run test suite
make test-coverage  # Generate coverage report
make clean          # Clean build artifacts
```

### Project Structure

```
fybrk/
├── cli/                   # Command-line interface
│   └── cmd/fybrk/         # Main CLI application
├── internal/              # Internal packages
│   ├── network/           # P2P networking and UPnP
│   ├── pairing/           # Device pairing system
│   ├── protocol/          # Universal sync protocol
│   ├── storage/           # Encryption, chunking, metadata
│   ├── sync/              # Sync engine and multi-device
│   └── watcher/           # File system monitoring
├── pkg/                   # Public API packages
│   ├── api/               # Cross-platform API
│   ├── fybrk/             # Main client library
│   └── types/             # Shared data types
└── integration_test.go    # Integration tests
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/storage
go test ./internal/sync
```

## Configuration

### Environment Variables

- `FYBRK_LOG_LEVEL`: Set logging level (debug, info, warn, error)
- `FYBRK_PORT`: Override default network port
- `FYBRK_CHUNK_SIZE`: Set custom chunk size (default: 1MB)

### File Structure

```
your-sync-folder/
├── .fybrk/
│   ├── key              # Encryption key (32 bytes)
│   └── metadata.db      # SQLite database
├── your-files.txt       # Your synced files
└── subdirectory/        # Subdirectories are synced too
    └── more-files.pdf
```

## Security Model

### Encryption

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Generation**: Cryptographically secure random 32-byte keys
- **Key Storage**: Local only, never transmitted
- **Integrity**: SHA-256 checksums for all data

### Network Security

- **Device Authentication**: QR code-based secure pairing
- **Message Encryption**: All P2P communication encrypted
- **No Plaintext**: Data never transmitted unencrypted
- **Zero Trust**: No central authority or servers

### Privacy Guarantees

- **Local Processing**: All encryption/decryption happens locally
- **No Telemetry**: No usage data collected or transmitted
- **No Accounts**: No registration or user accounts required
- **Open Source**: Full transparency of security implementation

## Performance

### Benchmarks

- **File Detection**: Sub-second change detection
- **Sync Speed**: Limited by network bandwidth, not CPU
- **Memory Usage**: ~10MB base + file buffers
- **Concurrent Files**: Handles thousands of files efficiently

### Optimization Features

- **Incremental Sync**: Only changed chunks are transferred
- **Deduplication**: Identical files share storage
- **Compression**: Optional compression for network transfer
- **Batch Operations**: Multiple file changes batched together

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with tests
4. Ensure all tests pass (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Quality Standards

- **Test Coverage**: New code must include comprehensive tests
- **Documentation**: Public APIs must be documented
- **Code Style**: Follow Go conventions and use `gofmt`
- **Security**: Security-related changes require extra review

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [fsnotify](https://github.com/fsnotify/fsnotify) for file system monitoring
- [testify](https://github.com/stretchr/testify) for testing framework
- Go standard library for cryptographic primitives

## Support

- **Documentation**: [fybrk.com](https://fybrk.com)
- **Issues**: [GitHub Issues](https://github.com/Fybrk/fybrk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Fybrk/fybrk/discussions)

---

**Fybrk: Your files, everywhere, private by design.**
