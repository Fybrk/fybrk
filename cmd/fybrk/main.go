package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Fybrk/fybrk/pkg/core"
	"github.com/Fybrk/fybrk/internal/config"
)

// Version is set at build time via ldflags
var Version = "dev"

func main() {
	// Initialize config on first run
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Warning: Could not load config: %v\n", err)
	}
	
	// Parse arguments with simple logic
	var target string

	if len(os.Args) == 1 {
		// No arguments - use current directory
		target = "."
	} else if len(os.Args) == 2 {
		target = os.Args[1]

		// Handle help requests
		if target == "help" || target == "-h" || target == "--help" {
			showUsage()
			return
		}

		// Handle pair URL request
		if target == "pair" || target == "--pair" {
			showPairURL(".")
			return
		}

		// Handle version request
		if target == "version" || target == "--version" || target == "-v" {
			fmt.Printf("fybrk version %s\n", Version)
			return
		}

		// Handle config request
		if target == "config" || target == "--config" {
			showConfig(cfg)
			return
		}
	} else {
		fmt.Println("Error: Too many arguments")
		showUsage()
		os.Exit(1)
	}

	// Determine action based on target format
	if core.IsValidPairURL(target) {
		// Join existing sync
		runJoin(target)
	} else {
		// Start new sync
		runStart(target)
	}
}

func runStart(syncPath string) {
	fmt.Printf("Starting Fybrk sync in: %s\n", syncPath)

	// Create Fybrk instance (auto-initializes everything)
	config := core.Config{SyncPath: syncPath}
	fybrk, err := core.New(config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer fybrk.Close()

	// Start sync engine
	if err := fybrk.StartSync(); err != nil {
		fmt.Printf("Error starting sync: %v\n", err)
		os.Exit(1)
	}

	// Generate pairing information
	pairData, err := fybrk.GeneratePairData()
	if err != nil {
		fmt.Printf("Error generating pair data: %v\n", err)
		os.Exit(1)
	}

	// Show compact pairing info
	fmt.Printf("Pair with: %s\n", pairData.URL)
	fmt.Printf("Server: %s\n", fybrk.GetServerAddress())
	fmt.Printf("Expires: %s\n", pairData.ExpiresAt.Format("15:04:05"))
	fmt.Println()
	fmt.Println("Syncing files in real-time...")
	fmt.Println("Press Ctrl+C to stop")

	// Keep running
	select {}
}

func runJoin(pairURL string) {
	fmt.Println("Joining sync from pair URL...")

	// Default to current directory for local sync
	localPath, _ := os.Getwd()
	localPath = filepath.Join(localPath, "fybrk-sync")

	// Join the sync
	fybrk, err := core.JoinFromPairData(pairURL, localPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer fybrk.Close()

	// Start sync engine
	if err := fybrk.StartSync(); err != nil {
		fmt.Printf("Error starting sync: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Syncing to: %s\n", fybrk.GetSyncPath())
	fmt.Println("Connected! Syncing files in real-time...")
	fmt.Println("Press Ctrl+C to stop")

	// Keep running
	select {}
}

func showPairURL(syncPath string) {
	fmt.Printf("Getting pair URL for: %s\n", syncPath)

	// Create Fybrk instance (auto-initializes everything)
	config := core.Config{SyncPath: syncPath}
	fybrk, err := core.New(config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer fybrk.Close()

	// Generate pairing information
	pairData, err := fybrk.GeneratePairData()
	if err != nil {
		fmt.Printf("Error generating pair data: %v\n", err)
		os.Exit(1)
	}

	// Show pairing info
	fmt.Printf("Pair with: %s\n", pairData.URL)
	fmt.Printf("Expires: %s\n", pairData.ExpiresAt.Format("15:04:05"))
	fmt.Println()
	fmt.Println("Share this URL with other devices to sync files.")
}

func showUsage() {
	fmt.Println("Fybrk - Your files, everywhere, private by design")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  fybrk                          # Start sync in current directory")
	fmt.Println("  fybrk /path/to/folder          # Start sync in specific directory")
	fmt.Println("  fybrk fybrk://pair?data=...    # Join existing sync")
	fmt.Println("  fybrk pair                     # Get pair URL for current directory")
	fmt.Println("  fybrk config                   # Show current configuration")
	fmt.Println("  fybrk version                  # Show version")
	fmt.Println("  fybrk help                     # Show this help")
	fmt.Println()
	fmt.Println("QUICK START - SYNC BETWEEN 2 DEVICES:")
	fmt.Println()
	fmt.Println("  Device 1 (has files to share):")
	fmt.Println("    1. fybrk ~/Documents")
	fmt.Println("    2. Copy the 'Pair with:' URL that appears")
	fmt.Println("    3. Send URL to Device 2 (text, email, etc.)")
	fmt.Println()
	fmt.Println("  Device 2 (wants to receive files):")
	fmt.Println("    1. fybrk 'fybrk://pair?key=...'  # Paste the URL from Device 1")
	fmt.Println("    2. Files sync automatically!")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  fybrk                          # Sync current directory")
	fmt.Println("  fybrk ~/Documents              # Sync Documents folder")
	fmt.Println("  fybrk ~/Photos                 # Sync Photos folder")
	fmt.Println("  fybrk pair                     # Get pair URL for current directory")
	fmt.Println("  fybrk version                  # Show version info")
	fmt.Println("  fybrk 'fybrk://pair?key=abc'   # Join from pair URL")
	fmt.Println()
	fmt.Println("WHAT HAPPENS WHEN YOU RUN FYBRK:")
	fmt.Println("  1. Fybrk scans files and starts watching for changes")
	fmt.Println("  2. WebSocket server starts for peer connections")
	fmt.Println("  3. Generates a pair URL for other devices to join")
	fmt.Println("  4. File events (create/modify/delete) sync in real-time")
	fmt.Println("  5. All changes are tracked in local SQLite database")
	fmt.Println()
	fmt.Println("FEATURES:")
	fmt.Println("  - Real-time 2-way file synchronization")
	fmt.Println("  - Hash-based deduplication (only sync content changes)")
	fmt.Println("  - Automatic conflict resolution")
	fmt.Println("  - Zero configuration required")
	fmt.Println("  - Cross-platform (Windows, macOS, Linux)")
	fmt.Println("  - No cloud servers - direct device-to-device + relay fallback")
	fmt.Println()
	fmt.Println("TROUBLESHOOTING:")
	fmt.Println("  - Files not syncing? Check both devices are running fybrk")
	fmt.Println("  - Connection issues? Relay servers provide internet fallback")
	fmt.Println("  - Custom relay? Edit ~/.fybrk/config.json")
	fmt.Println("  - Need help? Visit https://github.com/Fybrk/fybrk/issues")
}

func showConfig(cfg *config.Config) {
	configDir, _ := config.GetConfigDir()
	fmt.Printf("Fybrk Configuration\n")
	fmt.Printf("===================\n")
	fmt.Printf("Config file: %s/config.json\n", configDir)
	fmt.Printf("Device ID: %s\n", cfg.DeviceID)
	fmt.Printf("Relay enabled: %t\n", cfg.EnableRelay)
	fmt.Printf("Relay servers:\n")
	for _, server := range cfg.RelayServers {
		fmt.Printf("  - %s\n", server)
	}
	fmt.Printf("\nTo customize, edit the config file and restart fybrk.\n")
}
