package core

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileEvent represents a file system event
type FileEvent struct {
	Path      string
	Type      EventType
	Hash      string
	Size      int64
	ModTime   time.Time
	Timestamp time.Time
}

// EventType represents the type of file event
type EventType int

const (
	EventCreate EventType = iota
	EventModify
	EventDelete
)

// String returns the string representation of EventType
func (e EventType) String() string {
	switch e {
	case EventCreate:
		return "create"
	case EventModify:
		return "modify"
	case EventDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Watcher monitors file system changes
type Watcher struct {
	fybrk    *Fybrk
	watcher  *fsnotify.Watcher
	events   chan FileEvent
	done     chan bool
	scanning bool
}

// NewWatcher creates a new file system watcher
func NewWatcher(fybrk *Fybrk) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	w := &Watcher{
		fybrk:   fybrk,
		watcher: fsWatcher,
		events:  make(chan FileEvent, 100),
		done:    make(chan bool),
	}

	// Add sync directory to watcher
	if err := w.addDirectory(fybrk.syncPath); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("failed to watch directory: %w", err)
	}

	go w.watchLoop()
	return w, nil
}

// addDirectory recursively adds directories to watcher
func (w *Watcher) addDirectory(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .fybrk directory
		if strings.Contains(walkPath, ".fybrk") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return w.watcher.Add(walkPath)
		}
		return nil
	})
}

// watchLoop processes file system events
func (w *Watcher) watchLoop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)

		case <-w.done:
			return
		}
	}
}

// handleEvent processes a single file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Skip .fybrk files
	if strings.Contains(event.Name, ".fybrk") {
		return
	}

	// Skip temporary files
	if strings.HasSuffix(event.Name, "~") || strings.HasPrefix(filepath.Base(event.Name), ".") {
		return
	}

	relPath, err := filepath.Rel(w.fybrk.syncPath, event.Name)
	if err != nil {
		return
	}

	var fileEvent FileEvent
	fileEvent.Path = relPath
	fileEvent.Timestamp = time.Now()

	if event.Op&fsnotify.Remove == fsnotify.Remove {
		fileEvent.Type = EventDelete
		w.events <- fileEvent
		return
	}

	// Get file info
	info, err := os.Stat(event.Name)
	if err != nil {
		return // File might have been deleted
	}

	if info.IsDir() {
		// Add new directory to watcher
		w.watcher.Add(event.Name)
		return
	}

	fileEvent.Size = info.Size()
	fileEvent.ModTime = info.ModTime()

	// Calculate hash for content changes
	hash, err := w.calculateHash(event.Name)
	if err != nil {
		return
	}
	fileEvent.Hash = hash

	if event.Op&fsnotify.Create == fsnotify.Create {
		fileEvent.Type = EventCreate
	} else {
		fileEvent.Type = EventModify
	}

	w.events <- fileEvent
}

// calculateHash computes SHA256 hash of file
func (w *Watcher) calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// Events returns the events channel
func (w *Watcher) Events() <-chan FileEvent {
	return w.events
}

// InitialScan performs initial scan of sync directory
func (w *Watcher) InitialScan() error {
	w.scanning = true
	defer func() { w.scanning = false }()

	return filepath.Walk(w.fybrk.syncPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .fybrk directory
		if strings.Contains(path, ".fybrk") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(w.fybrk.syncPath, path)
		if err != nil {
			return err
		}

		hash, err := w.calculateHash(path)
		if err != nil {
			return err
		}

		// Store in database
		if err := w.fybrk.updateFileRecord(relPath, info.Size(), info.ModTime(), hash); err != nil {
			return err
		}

		return nil
	})
}

// Close stops the watcher
func (w *Watcher) Close() error {
	close(w.done)
	return w.watcher.Close()
}
