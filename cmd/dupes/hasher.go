//go:build darwin

package main

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cespare/xxhash/v2"
)

// bufPool reuses read buffers for streaming hash computation.
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, hashBufferSize)
		return &b
	},
}

// hashFilePartial reads the first partialHashSize bytes and returns an xxhash hex string.
// Used for fast rejection of same-size but different files.
func hashFilePartial(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, partialHashSize)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	h := xxhash.Sum64(buf[:n])
	return fmt.Sprintf("%016x", h), nil
}

// hashFileFull computes a streaming xxhash of the entire file.
// It verifies that the current size matches expectedSize to detect modifications during scan.
func hashFileFull(path string, expectedSize int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Re-check size before hashing to detect modification.
	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	if info.Size() != expectedSize {
		return "", fmt.Errorf("size changed for %s: expected %d, got %d", path, expectedSize, info.Size())
	}

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)

	h := xxhash.New()
	if _, err := io.CopyBuffer(h, f, *bufPtr); err != nil {
		return "", fmt.Errorf("hashing %s: %w", path, err)
	}

	return fmt.Sprintf("%016x", h.Sum64()), nil
}
