package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Fybrk/fybrk/pkg/fybrk"
)

func main() {
	var (
		syncPath = flag.String("path", "", "Path to sync directory")
		dbPath   = flag.String("db", "", "Path to metadata database")
		command  = flag.String("cmd", "sync", "Command to run: sync, scan, list")
	)
	flag.Parse()

	if *syncPath == "" {
		fmt.Println("Error: -path is required")
		flag.Usage()
		os.Exit(1)
	}

	if *dbPath == "" {
		*dbPath = filepath.Join(*syncPath, ".fybrk", "metadata.db")
	}

	// Ensure .fybrk directory exists
	fybrDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(fybrDir, 0755); err != nil {
		fmt.Printf("Error creating .fybrk directory: %v\n", err)
		os.Exit(1)
	}

	// Generate or load encryption key (simplified for MVP)
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
		SyncPath:  *syncPath,
		DBPath:    *dbPath,
		DeviceID:  "local-device", // Simplified for MVP
		ChunkSize: 1024 * 1024,    // 1MB chunks
		Key:       key,
	}

	client, err := fybrk.NewClient(config)
	if err != nil {
		fmt.Printf("Error initializing Fybrk client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	switch *command {
	case "sync":
		runSync(client, *syncPath)
	case "scan":
		runScan(client, *syncPath)
	case "list":
		runList(client)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		flag.Usage()
		os.Exit(1)
	}
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
