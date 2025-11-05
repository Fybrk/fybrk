package storage

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChunker(t *testing.T) {
	t.Run("default chunk size", func(t *testing.T) {
		chunker := NewChunker(0)
		assert.Equal(t, DefaultChunkSize, chunker.chunkSize)
	})

	t.Run("custom chunk size", func(t *testing.T) {
		chunker := NewChunker(2048)
		assert.Equal(t, 2048, chunker.chunkSize)
	})
}

func TestChunkFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("Hello, Fybrk! This is test data for chunking.")
	
	err := os.WriteFile(testFile, testData, 0644)
	require.NoError(t, err)

	t.Run("chunk small file", func(t *testing.T) {
		chunker := NewChunker(1024) // 1KB chunks
		chunks, err := chunker.ChunkFile(testFile)
		
		require.NoError(t, err)
		assert.Len(t, chunks, 1) // Small file should be one chunk
		
		// Verify chunk data
		chunk := chunks[0]
		assert.Equal(t, testData, chunk.Data)
		assert.Equal(t, int64(len(testData)), chunk.Size)
		assert.False(t, chunk.Encrypted)
		
		// Verify hash
		expectedHash := sha256.Sum256(testData)
		assert.Equal(t, expectedHash, chunk.Hash)
	})

	t.Run("chunk file with multiple chunks", func(t *testing.T) {
		chunker := NewChunker(10) // Very small chunks to force multiple
		chunks, err := chunker.ChunkFile(testFile)
		
		require.NoError(t, err)
		assert.Greater(t, len(chunks), 1) // Should have multiple chunks
		
		// Verify total size
		totalSize := int64(0)
		for _, chunk := range chunks {
			totalSize += chunk.Size
		}
		assert.Equal(t, int64(len(testData)), totalSize)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		chunker := NewChunker(1024)
		_, err := chunker.ChunkFile("/nonexistent/file.txt")
		assert.Error(t, err)
	})
}

func TestReassembleChunks(t *testing.T) {
	originalData := []byte("Test data for reassembly")
	chunker := NewChunker(10) // Small chunks
	
	// Create temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, originalData, 0644)
	require.NoError(t, err)
	
	// Chunk the file
	chunks, err := chunker.ChunkFile(testFile)
	require.NoError(t, err)
	
	// Reassemble
	reassembled, err := chunker.ReassembleChunks(chunks)
	require.NoError(t, err)
	
	assert.Equal(t, originalData, reassembled)
}

func TestChunkIntegrity(t *testing.T) {
	// Test that chunking and reassembling preserves data integrity
	testData := make([]byte, 5000) // 5KB of data
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "integrity_test.bin")
	err := os.WriteFile(testFile, testData, 0644)
	require.NoError(t, err)
	
	chunker := NewChunker(1024) // 1KB chunks
	
	// Chunk
	chunks, err := chunker.ChunkFile(testFile)
	require.NoError(t, err)
	
	// Reassemble
	reassembled, err := chunker.ReassembleChunks(chunks)
	require.NoError(t, err)
	
	// Verify integrity
	assert.Equal(t, testData, reassembled)
	
	// Verify each chunk has correct hash
	offset := 0
	for _, chunk := range chunks {
		expectedData := testData[offset : offset+int(chunk.Size)]
		expectedHash := sha256.Sum256(expectedData)
		assert.Equal(t, expectedHash, chunk.Hash)
		offset += int(chunk.Size)
	}
}
