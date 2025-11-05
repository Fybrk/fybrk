package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/Fybrk/fybrk/pkg/types"
)

var (
	ErrInvalidKeySize = errors.New("invalid key size")
	ErrDecryption     = errors.New("decryption failed")
)

type Encryptor struct {
	key []byte
}

func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 { // AES-256 requires 32-byte key
		return nil, ErrInvalidKeySize
	}
	return &Encryptor{key: key}, nil
}

// EncryptChunk encrypts a chunk's data using AES-GCM
func (e *Encryptor) EncryptChunk(chunk *types.Chunk) error {
	if chunk.Encrypted {
		return nil // Already encrypted
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nonce, nonce, chunk.Data, nil)
	chunk.Data = ciphertext
	chunk.Encrypted = true

	return nil
}

// DecryptChunk decrypts a chunk's data using AES-GCM
func (e *Encryptor) DecryptChunk(chunk *types.Chunk) error {
	if !chunk.Encrypted {
		return nil // Not encrypted
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonceSize := gcm.NonceSize()
	if len(chunk.Data) < nonceSize {
		return ErrDecryption
	}

	nonce, ciphertext := chunk.Data[:nonceSize], chunk.Data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ErrDecryption
	}

	chunk.Data = plaintext
	chunk.Encrypted = false

	return nil
}
