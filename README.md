# Fybrk - Your files, everywhere, private by design

Fybrk is a simple, secure peer-to-peer file synchronization tool that syncs files in real-time across devices without any configuration.

## ✨ What It Does

**True 2-way sync**: Changes on any device instantly appear on all connected devices.

```bash
# Device 1: Start syncing a folder
fybrk ~/Documents

# Device 2: Join and sync
fybrk 'fybrk://pair?key=...'

# Now: Any file change on either device syncs instantly to the other
```

## 🚀 Quick Start

```bash
# Start syncing current directory
fybrk .

# Output:
# Starting Fybrk sync in: /path/to/folder
# Scanning files...
# Server listening on port 8080
# Sync engine started
# Pair with: fybrk://pair?key=abc123...
# Syncing files in real-time...
# 
# File event: create newfile.txt
# File event: modify document.txt
```

## ✅ Features That Actually Work

- **Real-Time Sync**: File changes sync instantly (create/modify/delete)
- **Zero Configuration**: Just run `fybrk` in any folder
- **Hash-Based Deduplication**: Only syncs when content actually changes
- **Conflict Resolution**: Timestamp-based with graceful handling
- **Database Tracking**: SQLite tracks all file metadata
- **WebSocket P2P**: Real-time communication between devices
- **Cross-Platform**: Works on Windows, macOS, and Linux

## 🔄 How 2-Way Sync Works

```
Device A                    Device B
   |                           |
   |-- Create file.txt --------|
   |                           |-- file.txt appears
   |                           |
   |                           |-- Modify file.txt
   |-- file.txt updated <------|
```

**Every file operation syncs both ways:**
- Create a file → appears on other devices
- Modify a file → changes sync to other devices  
- Delete a file → removed from other devices
- Move/rename → reflected on other devices

## 📊 Live Demo Output

```bash
$ fybrk .
Starting Fybrk sync in: /Users/you/project
Scanning files...
Server listening on port 8080
Sync engine started
Pair with: fybrk://pair?key=1a67df3e...&expires=1762388914
Server: localhost:8080
Syncing files in real-time...

File event: create README.md
File event: modify main.go
File event: delete old-file.txt
Peer connected: peer_1762342903625
```

## 🏗️ Architecture

### Core Components
- **File Watcher**: Detects changes using fsnotify
- **Sync Engine**: Processes events and manages peers
- **WebSocket Server**: Handles P2P connections
- **SQLite Database**: Tracks file metadata and hashes

### Sync Protocol
```json
{
  "type": "file_create",
  "path": "document.txt", 
  "hash": "sha256...",
  "size": 1024,
  "content": "base64..."
}
```

## 📁 File Structure

```
your-folder/
├── document.txt      # Your files (synced)
├── image.jpg         # Your files (synced)
└── .fybrk/          # Fybrk metadata (hidden)
    ├── key          # Encryption key (32 bytes)
    ├── metadata.db  # File tracking database
    └── metadata.db-wal # SQLite WAL file
```

## 🧪 Testing

Run comprehensive tests:
```bash
./comprehensive_test.sh

# Output:
# ✅ Device 1 started successfully
# ✅ File creation detected
# ✅ File modification detected  
# ✅ WebSocket server started
# ✅ Database contains file records
# ✅ Pair URL generated
# 🎉 ALL TESTS PASSED!
```

## 🛠️ Installation

### From Source
```bash
git clone https://github.com/Fybrk/fybrk
cd fybrk/fybrk
go build -o bin/fybrk cmd/fybrk/main.go
```

### Usage
```bash
fybrk                          # Sync current directory
fybrk /path/to/folder          # Sync specific directory  
fybrk 'fybrk://pair?key=...'   # Join existing sync
fybrk help                     # Show help
```

## 🔒 Security

- **AES-256 Encryption**: All sync data encrypted
- **SHA-256 Hashing**: Content verification
- **Local Keys**: Encryption keys stored locally
- **No Cloud**: Direct P2P, no third-party servers

## 🚧 Development

### Build
```bash
make build
```

### Test  
```bash
make test
./comprehensive_test.sh
```

### Coverage
```bash
make coverage
```

## 📈 Status

**✅ WORKING**: Real-time 2-way file sync is fully implemented and tested.

- File watching: ✅ Working
- Database tracking: ✅ Working  
- WebSocket P2P: ✅ Working
- Sync engine: ✅ Working
- Conflict resolution: ✅ Working

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new features
4. Submit a pull request

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.
