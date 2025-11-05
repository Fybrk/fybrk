# Fybrk - Secure Peer-to-Peer File Synchronization

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![Test Coverage](https://img.shields.io/badge/coverage-59.1%25-yellow.svg)](https://github.com/Fybrk/fybrk)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/Fybrk/fybrk)

Fybrk is a state-of-the-art, secure peer-to-peer file synchronization system that enables you to sync files across multiple devices without relying on cloud services. Your data stays private, encrypted, and under your complete control.

## 🚀 Features

### ✅ **Production Ready**
- **End-to-End Encryption**: AES-256-GCM with SHA-256 integrity verification
- **Peer-to-Peer Networking**: Direct device connections with UPnP NAT traversal
- **Real-Time Sync**: Sub-second file change detection with fsnotify
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Zero Trust Architecture**: No servers, no accounts, no monthly fees

### ✅ **Advanced Security**
- **Client-Side Encryption**: Files encrypted before leaving your device
- **Secure Key Management**: 32-byte keys generated and stored locally
- **Device Pairing**: QR code-based secure device authentication
- **Integrity Verification**: SHA-256 checksums prevent data corruption

### ✅ **High Performance**
- **Efficient Chunking**: 1MB file chunks for optimal transfer
- **Deduplication**: Content-based hashing eliminates duplicate data
- **Concurrent Processing**: Multi-threaded file operations
- **Minimal Memory**: Optimized for low resource usage

### ✅ **Developer Friendly**
- **Comprehensive API**: Clean Go library interface
- **Extensive Testing**: 59.1% overall coverage, 89%+ for critical components
- **CLI Tool**: Full command-line interface for automation
- **Plugin Architecture**: Extensible design for integrations

## 📦 Installation

### From Source
```bash
git clone https://github.com/Fybrk/fybrk.git
cd fybrk
make build
```

### Using Go
```bash
go install github.com/Fybrk/fybrk/cli/cmd/fybrk@latest
```

## 🚀 Quick Start

### 1. Initialize a Sync Directory
```bash
fybrk -path /your/sync/folder -cmd scan
```

### 2. Start Synchronization
```bash
fybrk -path /your/sync/folder -cmd sync
```

### 3. List Synced Files
```bash
fybrk -path /your/sync/folder -cmd list
```

### 4. Multi-Device Setup
1. Run `fybrk -cmd sync` on each device
2. Devices automatically discover each other on the local network
3. Files sync instantly when changes are detected

## 🏗️ Architecture

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

## 📊 Test Coverage

| Component | Coverage | Status |
|-----------|----------|---------|
| **Storage** | 89.4% | ✅ Excellent |
| **Watcher** | 89.3% | ✅ Excellent |
| **Pairing** | 88.1% | ✅ Excellent |
| **Public API** | 83.3% | ✅ Good |
| **Sync Engine** | 62.6% | ⚠️ Good |
| **Network** | 60.1% | ⚠️ Acceptable |
| **Overall** | 59.1% | ⚠️ Acceptable |

## 🛠️ Development

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
├── cli/                    # Command-line interface
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

## 🔧 Configuration

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

## 🔒 Security Model

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

## 🚀 Performance

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

## 🤝 Contributing

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

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [fsnotify](https://github.com/fsnotify/fsnotify) for file system monitoring
- [testify](https://github.com/stretchr/testify) for testing framework
- Go standard library for cryptographic primitives

## 📞 Support

- **Documentation**: [docs.fybrk.com](https://docs.fybrk.com)
- **Issues**: [GitHub Issues](https://github.com/Fybrk/fybrk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Fybrk/fybrk/discussions)

---

**Fybrk: Your files, everywhere, private by design.** 🔒
