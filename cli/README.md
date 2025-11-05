# Fybrk CLI

Command line interface for Fybrk file synchronization.

## Quick Install

```bash
curl -sSL https://fybrk.com/install.sh | bash
```

## Manual Installation

**Download pre-built binary:**
```bash
# Linux AMD64
curl -L https://github.com/Fybrk/cli/releases/latest/download/fybrk-linux-amd64 -o fybrk

# Linux ARM64
curl -L https://github.com/Fybrk/cli/releases/latest/download/fybrk-linux-arm64 -o fybrk

# Linux x86 (32-bit)
curl -L https://github.com/Fybrk/cli/releases/latest/download/fybrk-linux-386 -o fybrk

# macOS AMD64
curl -L https://github.com/Fybrk/cli/releases/latest/download/fybrk-darwin-amd64 -o fybrk

# macOS ARM64 (Apple Silicon)
curl -L https://github.com/Fybrk/cli/releases/latest/download/fybrk-darwin-arm64 -o fybrk

# Windows AMD64
curl -L https://github.com/Fybrk/cli/releases/latest/download/fybrk-windows-amd64.exe -o fybrk.exe

# Windows x86 (32-bit)
curl -L https://github.com/Fybrk/cli/releases/latest/download/fybrk-windows-386.exe -o fybrk.exe

chmod +x fybrk
```

**Build from source:**
```bash
git clone https://github.com/Fybrk/cli.git
cd cli
make build
```

## Usage

**Start synchronization:**
```bash
fybrk -path /path/to/sync -cmd sync
```

**Scan for changes:**
```bash
fybrk -path /path/to/sync -cmd scan
```

**List tracked files:**
```bash
fybrk -path /path/to/sync -cmd list
```

## Options

- `-path` - Directory to synchronize (required)
- `-db` - Metadata database path (optional)
- `-cmd` - Command: sync, scan, list (default: sync)

## Examples

**Basic synchronization:**
```bash
fybrk -path ~/Documents -cmd sync
```

**Custom database location:**
```bash
fybrk -path ~/Documents -db ~/fybrk.db -cmd scan
```

**List all tracked files:**
```bash
fybrk -path ~/Documents -cmd list
```

## Multi-Device Setup

**Device 1:**
```bash
fybrk -path ~/Documents -cmd sync
# Starts P2P discovery automatically
```

**Device 2:**
```bash
fybrk -path ~/Documents -cmd sync  
# Discovers and syncs with Device 1 automatically
```

No configuration needed - devices discover each other on the local network.

## Configuration

The CLI automatically creates a `.fybrk` directory in your sync path:

```
~/Documents/
├── .fybrk/
│   ├── metadata.db    # File tracking database
│   └── key           # Encryption key
└── [your files]
```

## Development

**Build:**
```bash
make build
```

**Test:**
```bash
make test
```

**Clean:**
```bash
make clean
```

## Dependencies

- [Fybrk Core](https://github.com/Fybrk/core) - Synchronization engine

## Contributing

1. Fork the repository
2. Make your changes
3. Test thoroughly
4. Submit a pull request

## License

MIT
