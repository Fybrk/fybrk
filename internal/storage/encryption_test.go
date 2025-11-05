package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/Fybrk/fybrk/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

func TestNewEncryptor(t *testing.T) {
	t.Run("valid key size", func(t *testing.T) {
		key := generateTestKey()
		encryptor, err := NewEncryptor(key)
		
		require.NoError(t, err)
		assert.NotNil(t, encryptor)
		assert.Equal(t, key, encryptor.key)
	})

	t.Run("invalid key size", func(t *testing.T) {
		invalidKey := make([]byte, 16) // Too short
		_, err := NewEncryptor(invalidKey)
		
		assert.Equal(t, ErrInvalidKeySize, err)
	})
}

func TestEncryptChunk(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	t.Run("encrypt unencrypted chunk", func(t *testing.T) {
		originalData := []byte("test data for encryption")
		hash := sha256.Sum256(originalData)
		
		chunk := &types.Chunk{
			Hash:      hash,
			Data:      originalData,
			Size:      int64(len(originalData)),
			Encrypted: false,
			CreatedAt: time.Now(),
		}

		err := encryptor.EncryptChunk(chunk)
		require.NoError(t, err)
		
		assert.True(t, chunk.Encrypted)
		assert.NotEqual(t, originalData, chunk.Data) // Data should be different
		assert.Greater(t, len(chunk.Data), len(originalData)) // Encrypted data is larger (nonce + ciphertext)
	})

	t.Run("encrypt already encrypted chunk", func(t *testing.T) {
		originalData := []byte("already encrypted")
		hash := sha256.Sum256(originalData)
		
		chunk := &types.Chunk{
			Hash:      hash,
			Data:      originalData,
			Size:      int64(len(originalData)),
			Encrypted: true, // Already encrypted
			CreatedAt: time.Now(),
		}

		err := encryptor.EncryptChunk(chunk)
		require.NoError(t, err)
		
		assert.True(t, chunk.Encrypted)
		assert.Equal(t, originalData, chunk.Data) // Data should remain unchanged
	})
}

func TestDecryptChunk(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	t.Run("decrypt encrypted chunk", func(t *testing.T) {
		originalData := []byte("test data for decryption")
		hash := sha256.Sum256(originalData)
		
		chunk := &types.Chunk{
			Hash:      hash,
			Data:      originalData,
			Size:      int64(len(originalData)),
			Encrypted: false,
			CreatedAt: time.Now(),
		}

		// First encrypt
		err := encryptor.EncryptChunk(chunk)
		require.NoError(t, err)
		assert.True(t, chunk.Encrypted)
		
		encryptedData := make([]byte, len(chunk.Data))
		copy(encryptedData, chunk.Data)

		// Then decrypt
		err = encryptor.DecryptChunk(chunk)
		require.NoError(t, err)
		
		assert.False(t, chunk.Encrypted)
		assert.Equal(t, originalData, chunk.Data)
		assert.NotEqual(t, encryptedData, chunk.Data)
	})

	t.Run("decrypt unencrypted chunk", func(t *testing.T) {
		originalData := []byte("not encrypted")
		hash := sha256.Sum256(originalData)
		
		chunk := &types.Chunk{
			Hash:      hash,
			Data:      originalData,
			Size:      int64(len(originalData)),
			Encrypted: false,
			CreatedAt: time.Now(),
		}

		err := encryptor.DecryptChunk(chunk)
		require.NoError(t, err)
		
		assert.False(t, chunk.Encrypted)
		assert.Equal(t, originalData, chunk.Data)
	})

	t.Run("decrypt with wrong key", func(t *testing.T) {
		originalData := []byte("test data")
		hash := sha256.Sum256(originalData)
		
		chunk := &types.Chunk{
			Hash:      hash,
			Data:      originalData,
			Size:      int64(len(originalData)),
			Encrypted: false,
			CreatedAt: time.Now(),
		}

		// Encrypt with first key
		err := encryptor.EncryptChunk(chunk)
		require.NoError(t, err)

		// Try to decrypt with different key
		wrongKey := generateTestKey()
		wrongEncryptor, err := NewEncryptor(wrongKey)
		require.NoError(t, err)

		err = wrongEncryptor.DecryptChunk(chunk)
		assert.Equal(t, ErrDecryption, err)
	})

	t.Run("decrypt malformed data", func(t *testing.T) {
		chunk := &types.Chunk{
			Data:      []byte("too short"), // Not enough data for nonce
			Encrypted: true,
		}

		err := encryptor.DecryptChunk(chunk)
		assert.Equal(t, ErrDecryption, err)
	})
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := generateTestKey()
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	testCases := [][]byte{
		[]byte("short"),
		[]byte("medium length test data"),
		[]byte("very long test data that spans multiple lines and contains various characters: !@#$%^&*()_+-=[]{}|;':\",./<>?"),
		make([]byte, 1024), // 1KB of zeros
	}

	for i, originalData := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			hash := sha256.Sum256(originalData)
			
			chunk := &types.Chunk{
				Hash:      hash,
				Data:      originalData,
				Size:      int64(len(originalData)),
				Encrypted: false,
				CreatedAt: time.Now(),
			}

			// Make a copy of original data
			originalCopy := make([]byte, len(originalData))
			copy(originalCopy, originalData)

			// Encrypt
			err := encryptor.EncryptChunk(chunk)
			require.NoError(t, err)
			assert.True(t, chunk.Encrypted)

			// Decrypt
			err = encryptor.DecryptChunk(chunk)
			require.NoError(t, err)
			assert.False(t, chunk.Encrypted)

			// Verify data integrity
			assert.Equal(t, originalCopy, chunk.Data)
		})
	}
}
