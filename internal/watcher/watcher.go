package watcher

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type FileEvent struct {
	Path      string
	Operation string // "create", "write", "remove", "rename"
}

type FileWatcher struct {
	watcher   *fsnotify.Watcher
	events    chan FileEvent
	errors    chan error
	done      chan bool
	watchDirs map[string]bool
	mu        sync.RWMutex
}

func NewFileWatcher() (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw := &FileWatcher{
		watcher:   watcher,
		events:    make(chan FileEvent, 100),
		errors:    make(chan error, 10),
		done:      make(chan bool),
		watchDirs: make(map[string]bool),
	}

	go fw.run()
	return fw, nil
}

func (fw *FileWatcher) AddPath(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// Add directory and all subdirectories
		return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := fw.watcher.Add(walkPath); err != nil {
					return err
				}
				fw.watchDirs[walkPath] = true
			}
			return nil
		})
	} else {
		// Add parent directory for file
		dir := filepath.Dir(path)
		if !fw.watchDirs[dir] {
			if err := fw.watcher.Add(dir); err != nil {
				return err
			}
			fw.watchDirs[dir] = true
		}
	}

	return nil
}

func (fw *FileWatcher) RemovePath(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	err := fw.watcher.Remove(path)
	if err == nil {
		delete(fw.watchDirs, path)
	}
	return err
}

func (fw *FileWatcher) Events() <-chan FileEvent {
	return fw.events
}

func (fw *FileWatcher) Errors() <-chan error {
	return fw.errors
}

func (fw *FileWatcher) Close() error {
	close(fw.done)
	return fw.watcher.Close()
}

func (fw *FileWatcher) run() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			select {
			case fw.errors <- err:
			default:
				// Error channel full, drop error
			}

		case <-fw.done:
			return
		}
	}
}

func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	var operation string

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = "create"
		// If a new directory was created, watch it
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			fw.AddPath(event.Name)
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = "write"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = "remove"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		operation = "rename"
	default:
		return // Ignore other operations
	}

	fileEvent := FileEvent{
		Path:      event.Name,
		Operation: operation,
	}

	select {
	case fw.events <- fileEvent:
	default:
		// Event channel full, drop event
	}
}
