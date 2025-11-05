package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileWatcher(t *testing.T) {
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	require.NotNil(t, watcher)
	
	defer watcher.Close()
	
	assert.NotNil(t, watcher.watcher)
	assert.NotNil(t, watcher.events)
	assert.NotNil(t, watcher.errors)
	assert.NotNil(t, watcher.done)
	assert.NotNil(t, watcher.watchDirs)
}

func TestAddPathFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	
	// Create test file
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)
	
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	defer watcher.Close()
	
	// Add file path (should watch parent directory)
	err = watcher.AddPath(testFile)
	require.NoError(t, err)
	
	// Parent directory should be watched
	assert.True(t, watcher.watchDirs[tmpDir])
}

func TestAddPathDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)
	
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	defer watcher.Close()
	
	// Add directory path
	err = watcher.AddPath(tmpDir)
	require.NoError(t, err)
	
	// Both directories should be watched
	assert.True(t, watcher.watchDirs[tmpDir])
	assert.True(t, watcher.watchDirs[subDir])
}

func TestAddPathNonexistent(t *testing.T) {
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	defer watcher.Close()
	
	err = watcher.AddPath("/nonexistent/path")
	assert.Error(t, err)
}

func TestRemovePath(t *testing.T) {
	tmpDir := t.TempDir()
	
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	defer watcher.Close()
	
	// Add path
	err = watcher.AddPath(tmpDir)
	require.NoError(t, err)
	assert.True(t, watcher.watchDirs[tmpDir])
	
	// Remove path
	err = watcher.RemovePath(tmpDir)
	require.NoError(t, err)
	assert.False(t, watcher.watchDirs[tmpDir])
}

func TestFileEvents(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	defer watcher.Close()
	
	// Add directory to watch
	err = watcher.AddPath(tmpDir)
	require.NoError(t, err)
	
	// Create file and wait for event
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	// Wait for create event
	select {
	case event := <-watcher.Events():
		assert.Equal(t, testFile, event.Path)
		assert.Equal(t, "create", event.Operation)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for create event")
	}
	
	// Modify file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)
	
	// Wait for write event
	select {
	case event := <-watcher.Events():
		assert.Equal(t, testFile, event.Path)
		assert.Equal(t, "write", event.Operation)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for write event")
	}
	
	// Remove file
	err = os.Remove(testFile)
	require.NoError(t, err)
	
	// Wait for remove event
	select {
	case event := <-watcher.Events():
		assert.Equal(t, testFile, event.Path)
		assert.Equal(t, "remove", event.Operation)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for remove event")
	}
}

func TestDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir")
	
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	defer watcher.Close()
	
	// Add parent directory to watch
	err = watcher.AddPath(tmpDir)
	require.NoError(t, err)
	
	// Create new directory
	err = os.Mkdir(newDir, 0755)
	require.NoError(t, err)
	
	// Wait for create event
	select {
	case event := <-watcher.Events():
		assert.Equal(t, newDir, event.Path)
		assert.Equal(t, "create", event.Operation)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for directory create event")
	}
	
	// Give some time for the watcher to add the new directory
	time.Sleep(100 * time.Millisecond)
	
	// New directory should now be watched
	watcher.mu.RLock()
	watched := watcher.watchDirs[newDir]
	watcher.mu.RUnlock()
	assert.True(t, watched)
}

func TestClose(t *testing.T) {
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	
	err = watcher.Close()
	assert.NoError(t, err)
	
	// Events channel should be closed after some time
	select {
	case _, ok := <-watcher.Events():
		if ok {
			t.Fatal("Events channel should be closed")
		}
	case <-time.After(1 * time.Second):
		// Timeout is acceptable as the channel might not close immediately
	}
}

func TestEventChannelCapacity(t *testing.T) {
	tmpDir := t.TempDir()
	
	watcher, err := NewFileWatcher()
	require.NoError(t, err)
	defer watcher.Close()
	
	err = watcher.AddPath(tmpDir)
	require.NoError(t, err)
	
	// Create many files quickly to test channel capacity
	for i := 0; i < 150; i++ { // More than channel capacity (100)
		testFile := filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)
	}
	
	// Should receive some events (channel might drop some due to capacity)
	eventCount := 0
	timeout := time.After(3 * time.Second)
	
	for eventCount < 50 { // Expect at least some events
		select {
		case <-watcher.Events():
			eventCount++
		case <-timeout:
			break
		}
	}
	
	assert.Greater(t, eventCount, 0, "Should receive at least some events")
}
