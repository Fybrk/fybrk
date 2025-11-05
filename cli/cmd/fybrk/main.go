package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Fybrk/fybrk/pkg/fybrk"
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	var syncPath, command string

	// Parse arguments - support both formats:
	// fybrk /path command
	// fybrk command /path
	if len(os.Args) == 3 {
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
		// Single argument - could be help request or path with default sync
		arg := os.Args[1]
		if arg == "help" || arg == "-h" || arg == "--help" {
			showUsage()
			os.Exit(0)
		}
		// Default to sync command
		syncPath = arg
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

	// Validate path
	if syncPath == "" {
		fmt.Println("Error: Path is required\n")
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
	case "scan":
		runScan(client, syncPath)
	case "list":
		runList(client)
	}
}

func isValidCommand(cmd string) bool {
	validCommands := []string{"sync", "scan", "list"}
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
	fmt.Println("  fybrk <path> [command]     # Path first, then command")
	fmt.Println("  fybrk <command> <path>     # Command first, then path")
	fmt.Println("  fybrk <path>               # Default to sync command")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  scan    Initialize directory for sync (first-time setup)")
	fmt.Println("  sync    Start real-time synchronization (default)")
	fmt.Println("  list    List all tracked files and their status")
	fmt.Println()
	fmt.Println("WORKFLOW:")
	fmt.Println("  1. fybrk /path/to/folder scan    # First time: scan and encrypt files")
	fmt.Println("  2. fybrk /path/to/folder sync    # Start syncing with other devices")
	fmt.Println("  3. fybrk /path/to/folder list    # Check what files are tracked")
	fmt.Println()
	fmt.Println("WHAT EACH COMMAND DOES:")
	fmt.Println("  scan - Creates .fybrk folder, generates encryption key, scans all files")
	fmt.Println("  sync - Monitors for file changes and syncs with other devices")
	fmt.Println("  list - Shows all files being tracked with version info")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  fybrk ~/Documents scan         # Set up ~/Documents for syncing")
	fmt.Println("  fybrk ~/Documents              # Start syncing ~/Documents")
	fmt.Println("  fybrk list ~/Documents         # See what's being synced")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  help, -h, --help              Show this help message")
	fmt.Println()
	fmt.Println("FIRST TIME SETUP:")
	fmt.Println("  Run 'scan' on a directory to initialize it for syncing.")
	fmt.Println("  This creates a .fybrk folder with:")
	fmt.Println("  - metadata.db (SQLite database with file info)")
	fmt.Println("  - key (32-byte encryption key)")
	fmt.Println()
	fmt.Println("MULTI-DEVICE SYNC:")
	fmt.Println("  After scanning, run 'sync' on each device.")
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
