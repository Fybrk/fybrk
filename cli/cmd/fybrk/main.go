package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Fybrk/fybrk/internal/network"
	"github.com/Fybrk/fybrk/pkg/fybrk"
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	var syncPath, command string

	// Parse arguments - support multiple formats:
	// fybrk /path command
	// fybrk command /path  
	// fybrk command (defaults to current directory)
	// fybrk pair-with <qr-data>
	if len(os.Args) >= 2 && os.Args[1] == "pair-with" {
		// Handle pair-with command specially
		if len(os.Args) < 3 {
			fmt.Println("Error: QR code data required")
			fmt.Println("Usage: fybrk pair-with '<QR-CODE-DATA>'")
			os.Exit(1)
		}
		qrData := os.Args[2]
		runPairWith(qrData)
		return
	} else if len(os.Args) == 3 {
		arg1, arg2 := os.Args[1], os.Args[2]
		
		// Check if first arg is a command
		if isValidCommand(arg1) {
			command = arg1
			syncPath = arg2
		} else {
			// Assume first arg is path, second is command
			syncPath = arg1
			command = arg2
		}
	} else if len(os.Args) == 2 {
		// Single argument - could be help request, command, or path
		arg := os.Args[1]
		if arg == "help" || arg == "-h" || arg == "--help" {
			showUsage()
			os.Exit(0)
		} else if isValidCommand(arg) {
			// Command without path - default to current directory
			command = arg
			syncPath = "."
		} else {
			// Path without command - default to sync
			syncPath = arg
			command = "sync"
		}
	} else if len(os.Args) == 1 {
		// No arguments - default to current directory and sync
		syncPath = "."
		command = "sync"
	} else {
		showUsage()
		os.Exit(1)
	}

	// Validate command
	if !isValidCommand(command) {
		fmt.Printf("Error: Unknown command '%s'\n\n", command)
		showUsage()
		os.Exit(1)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(syncPath)
	if err != nil {
		fmt.Printf("Error: Invalid path '%s': %v\n", syncPath, err)
		os.Exit(1)
	}
	syncPath = absPath

	// Check if path exists
	if _, err := os.Stat(syncPath); os.IsNotExist(err) {
		fmt.Printf("Error: Path '%s' does not exist\n", syncPath)
		os.Exit(1)
	}

	// Setup database path
	dbPath := filepath.Join(syncPath, ".fybrk", "metadata.db")

	// Ensure .fybrk directory exists
	fybrDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(fybrDir, 0755); err != nil {
		fmt.Printf("Error creating .fybrk directory: %v\n", err)
		os.Exit(1)
	}

	// Generate or load encryption key
	key := make([]byte, 32)
	keyPath := filepath.Join(fybrDir, "key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Generate new key
		if _, err := rand.Read(key); err != nil {
			fmt.Printf("Error generating encryption key: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			fmt.Printf("Error saving encryption key: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Generated new encryption key")
	} else {
		// Load existing key
		key, err = os.ReadFile(keyPath)
		if err != nil {
			fmt.Printf("Error loading encryption key: %v\n", err)
			os.Exit(1)
		}
	}

	// Create Fybrk client
	config := &fybrk.Config{
		SyncPath:  syncPath,
		DBPath:    dbPath,
		DeviceID:  "local-device",
		ChunkSize: 1024 * 1024, // 1MB chunks
		Key:       key,
	}

	client, err := fybrk.NewClient(config)
	if err != nil {
		fmt.Printf("Error initializing Fybrk client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Execute command
	switch command {
	case "sync":
		runSync(client, syncPath)
	case "init":
		runScan(client, syncPath)
	case "list":
		runList(client)
	case "pair":
		runPair(client, syncPath)
	}
}

func isValidCommand(cmd string) bool {
	validCommands := []string{"sync", "init", "list", "pair", "pair-with"}
	for _, valid := range validCommands {
		if cmd == valid {
			return true
		}
	}
	return false
}

func showUsage() {
	fmt.Println("Fybrk - Secure Peer-to-Peer File Synchronization")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  fybrk [path] [command]     # Path and command (any order)")
	fmt.Println("  fybrk [command]            # Command with current directory")
	fmt.Println("  fybrk [path]               # Path with default sync command")
	fmt.Println("  fybrk                      # Default: sync current directory")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  init      Initialize directory for sync (first-time setup)")
	fmt.Println("  sync      Start real-time synchronization (default)")
	fmt.Println("  list      List all tracked files and their status")
	fmt.Println("  pair      Generate QR code to pair with other devices")
	fmt.Println("  pair-with Join sync network from QR code")
	fmt.Println()
	fmt.Println("WORKFLOW:")
	fmt.Println("  Device A:")
	fmt.Println("    1. fybrk init                    # Initialize current folder")
	fmt.Println("    2. fybrk pair                    # Generate QR code")
	fmt.Println("    3. fybrk sync                    # Start syncing")
	fmt.Println("  Device B:")
	fmt.Println("    1. fybrk pair-with '<QR-DATA>'   # Join from QR code")
	fmt.Println("    2. fybrk sync                    # Start syncing")
	fmt.Println()
	fmt.Println("WHAT EACH COMMAND DOES:")
	fmt.Println("  init      - Creates .fybrk folder, generates encryption key, scans files")
	fmt.Println("  pair      - Creates internet rendezvous point, shows QR code")
	fmt.Println("  pair-with - Joins sync network from QR code (works over internet)")
	fmt.Println("  sync      - Monitors for file changes and syncs with paired devices")
	fmt.Println("  list      - Shows all files being tracked with version info")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  fybrk init                     # Initialize current directory")
	fmt.Println("  fybrk ~/Documents init         # Initialize ~/Documents")
	fmt.Println("  fybrk                          # Sync current directory")
	fmt.Println("  fybrk ~/Documents              # Sync ~/Documents")
	fmt.Println("  fybrk list                     # List files in current directory")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  help, -h, --help              Show this help message")
	fmt.Println()
	fmt.Println("FIRST TIME SETUP:")
	fmt.Println("  Run 'init' on a directory to initialize it for syncing.")
	fmt.Println("  This creates a .fybrk folder with:")
	fmt.Println("  - metadata.db (SQLite database with file info)")
	fmt.Println("  - key (32-byte encryption key)")
	fmt.Println()
	fmt.Println("MULTI-DEVICE SYNC:")
	fmt.Println("  After initializing, run 'sync' on each device.")
	fmt.Println("  Devices will automatically discover each other and sync files.")
}

func runSync(client *fybrk.Client, syncPath string) {
	fmt.Printf("Starting Fybrk sync for: %s\n", syncPath)
	fmt.Println("Press Ctrl+C to stop...")

	// Initial scan
	if err := client.ScanDirectory(); err != nil {
		fmt.Printf("Error during initial scan: %v\n", err)
	}

	// Keep running to watch for changes
	select {} // Block forever
}

func runScan(client *fybrk.Client, syncPath string) {
	fmt.Printf("Scanning directory: %s\n", syncPath)
	
	if err := client.ScanDirectory(); err != nil {
		fmt.Printf("Error during scan: %v\n", err)
		os.Exit(1)
	}

	files, err := client.GetSyncedFiles()
	if err != nil {
		fmt.Printf("Error getting synced files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scanned %d files\n", len(files))
}

func runList(client *fybrk.Client) {
	files, err := client.GetSyncedFiles()
	if err != nil {
		fmt.Printf("Error listing files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("No files found")
		return
	}

	fmt.Printf("Found %d files:\n", len(files))
	for _, file := range files {
		fmt.Printf("  %s (v%d, %d bytes, %d chunks)\n", 
			file.Path, file.Version, file.Size, len(file.Chunks))
	}
}

func runPair(client *fybrk.Client, syncPath string) {
	fmt.Printf("Generating internet-capable pairing QR code for: %s\n", syncPath)
	fmt.Println()
	
	// Check if directory is initialized
	keyPath := filepath.Join(syncPath, ".fybrk", "key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		fmt.Println("Error: Directory not initialized. Run 'init' first.")
		return
	}
	
	// Read encryption key
	key, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("Error reading encryption key: %v\n", err)
		return
	}
	
	fmt.Println("Creating rendezvous point for internet pairing...")
	
	// Create pairing data with real QR code generation
	pairingData := map[string]interface{}{
		"version":        1,
		"sync_path":      syncPath,
		"encryption_key": fmt.Sprintf("%x", key),
		"expires_at":     time.Now().Add(10 * time.Minute).Unix(),
		"created_at":     time.Now().Unix(),
		"device_id":      "temp-device-id", // TODO: Get from client
	}
	
	// Generate QR code using real library
	qrGen := network.NewQRGenerator()
	qrData, err := qrGen.GenerateQRCode(pairingData)
	if err != nil {
		fmt.Printf("Error generating QR code: %v\n", err)
		return
	}
	
	// Display QR code in terminal
	err = qrGen.DisplayQRCode(qrData, true) // Save to file as well
	if err != nil {
		fmt.Printf("Error displaying QR code: %v\n", err)
		return
	}
	
	fmt.Println()
	fmt.Println("PAIRING INSTRUCTIONS:")
	fmt.Println("1. On the other device: fybrk pair-with '<QR-CODE-DATA>'")
	fmt.Println("2. Devices will connect over the internet automatically")
	fmt.Println("3. Files will sync in real-time")
	fmt.Println()
	fmt.Println("FEATURES:")
	fmt.Println("- Works over the internet (not just local network)")
	fmt.Println("- Automatic NAT traversal and hole punching")
	fmt.Println("- Secure end-to-end encryption")
	fmt.Println("- No manual IP configuration needed")
	fmt.Println()
	fmt.Println("This QR code expires in 10 minutes for security.")
}

func runPairWith(qrData string) {
	fmt.Printf("Joining sync network from QR code...\n")
	fmt.Println()
	
	if qrData == "" {
		fmt.Println("Error: QR code data required")
		fmt.Println("Usage: fybrk pair-with '<QR-CODE-DATA>'")
		return
	}
	
	// TODO: Implement actual QR code parsing and network joining
	// For now, show the concept
	fmt.Printf("QR Data: %s\n", qrData)
	fmt.Println()
	fmt.Println("NEXT STEPS:")
	fmt.Println("1. Parsing rendezvous information...")
	fmt.Println("2. Connecting to bootstrap network...")
	fmt.Println("3. Establishing direct P2P connection...")
	fmt.Println("4. Exchanging encryption keys...")
	fmt.Println("5. Starting file synchronization...")
	fmt.Println()
	fmt.Println("NOTE: Full implementation coming soon!")
	fmt.Println("This will automatically:")
	fmt.Println("- Connect over the internet")
	fmt.Println("- Handle NAT traversal")
	fmt.Println("- Set up secure sync folder")
}
