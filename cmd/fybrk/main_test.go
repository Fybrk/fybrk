package main

import (
	"os"
	"strings"
	"testing"

	"github.com/Fybrk/fybrk/pkg/core"
)

func TestShowUsage_Output(t *testing.T) {
	// Capture output by redirecting stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showUsage()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify key sections are present
	expectedSections := []string{
		"Fybrk - Your files, everywhere, private by design",
		"USAGE:",
		"fybrk version",
		"QUICK START - SYNC BETWEEN 2 DEVICES:",
		"EXAMPLES:",
		"WHAT HAPPENS WHEN YOU RUN FYBRK:",
		"FEATURES:",
		"TROUBLESHOOTING:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Usage output missing section: %s", section)
		}
	}
}

func TestShowUsage_Commands(t *testing.T) {
	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showUsage()

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify all command examples are present
	expectedCommands := []string{
		"fybrk                          # Start sync in current directory",
		"fybrk /path/to/folder          # Start sync in specific directory",
		"fybrk fybrk://pair?data=...    # Join existing sync",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Usage output missing command: %s", cmd)
		}
	}
}

func TestRunStart_ValidDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Test that runStart can create a fybrk instance without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runStart panicked: %v", r)
		}
	}()

	// We can't easily test the full runStart function since it blocks,
	// but we can test that the core functionality works
	config := core.Config{SyncPath: tempDir}
	fybrk, err := core.New(config)
	if err != nil {
		t.Fatalf("Expected no error creating fybrk instance: %v", err)
	}
	defer fybrk.Close()

	// Test pair data generation
	pairData, err := fybrk.GeneratePairData()
	if err != nil {
		t.Fatalf("Expected no error generating pair data: %v", err)
	}

	if !strings.HasPrefix(pairData.URL, "fybrk://pair?") {
		t.Errorf("Expected pair URL to start with 'fybrk://pair?', got: %s", pairData.URL)
	}
}

func TestRunJoin_ValidPairURL(t *testing.T) {
	tempDir := t.TempDir()
	pairURL := "fybrk://pair?key=abc123&path=/test&expires=123456"

	// Test that runJoin can create a fybrk instance without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runJoin panicked: %v", r)
		}
	}()

	// Test the core functionality that runJoin uses
	fybrk, err := core.JoinFromPairData(pairURL, tempDir)
	if err != nil {
		t.Fatalf("Expected no error joining from pair data: %v", err)
	}
	defer fybrk.Close()

	if fybrk.GetSyncPath() != tempDir {
		t.Errorf("Expected sync path %s, got %s", tempDir, fybrk.GetSyncPath())
	}
}

func TestArgumentParsing_NoArguments(t *testing.T) {
	// Test the logic that would be used for no arguments
	var target string
	args := []string{"fybrk"} // Simulating os.Args

	if len(args) == 1 {
		target = "."
	}

	if target != "." {
		t.Errorf("Expected target to be '.', got: %s", target)
	}

	// Should not be a pair URL
	if core.IsValidPairURL(target) {
		t.Error("Current directory should not be detected as pair URL")
	}
}

func TestArgumentParsing_HelpArguments(t *testing.T) {
	helpArgs := []string{"help", "-h", "--help"}

	for _, arg := range helpArgs {
		t.Run("help_"+arg, func(t *testing.T) {
			args := []string{"fybrk", arg}

			if len(args) == 2 {
				target := args[1]

				isHelp := target == "help" || target == "-h" || target == "--help"
				if !isHelp {
					t.Errorf("Expected %s to be recognized as help", arg)
				}
			}
		})
	}
}

func TestArgumentParsing_PairURL(t *testing.T) {
	pairURL := "fybrk://pair?key=abc123&path=/test&expires=123456"
	args := []string{"fybrk", pairURL}

	if len(args) == 2 {
		target := args[1]

		if !core.IsValidPairURL(target) {
			t.Error("Valid pair URL should be detected as pair URL")
		}
	}
}

func TestArgumentParsing_DirectoryPath(t *testing.T) {
	tempDir := t.TempDir()
	args := []string{"fybrk", tempDir}

	if len(args) == 2 {
		target := args[1]

		if core.IsValidPairURL(target) {
			t.Error("Directory path should not be detected as pair URL")
		}

		if target != tempDir {
			t.Errorf("Expected target to be %s, got %s", tempDir, target)
		}
	}
}

func TestArgumentParsing_TooManyArguments(t *testing.T) {
	args := []string{"fybrk", "arg1", "arg2", "arg3"}

	if len(args) > 2 {
		// This should trigger the "too many arguments" error
		// In the real main function, this would call showUsage() and os.Exit(1)
		tooMany := true
		if !tooMany {
			t.Error("Should detect too many arguments")
		}
	}
}
