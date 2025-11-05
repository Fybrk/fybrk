package storage

import (
	"crypto/sha256"
	"io"
	"os"
	"time"

	"github.com/Fybrk/fybrk/pkg/types"
)

const DefaultChunkSize = 1024 * 1024 // 1MB chunks

type Chunker struct {
	chunkSize int
}

func NewChunker(chunkSize int) *Chunker {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	return &Chunker{chunkSize: chunkSize}
}

// ChunkFile splits a file into encrypted chunks
func (c *Chunker) ChunkFile(filePath string) ([]types.Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunks []types.Chunk
	buffer := make([]byte, c.chunkSize)

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		chunkData := make([]byte, n)
		copy(chunkData, buffer[:n])
		hash := sha256.Sum256(chunkData)

		chunk := types.Chunk{
			Hash:      hash,
			Data:      chunkData,
			Size:      int64(n),
			Encrypted: false,
			CreatedAt: time.Now(),
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// ReassembleChunks combines chunks back into file data
func (c *Chunker) ReassembleChunks(chunks []types.Chunk) ([]byte, error) {
	var result []byte
	for _, chunk := range chunks {
		result = append(result, chunk.Data...)
	}
	return result, nil
}
