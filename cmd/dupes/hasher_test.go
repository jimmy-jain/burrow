//go:build darwin

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFilePartial_SamePrefix(t *testing.T) {
	dir := t.TempDir()

	// Two files with identical first 4KB but different tails.
	data := make([]byte, 8192)
	for i := range data {
		data[i] = byte(i % 256)
	}

	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")

	if err := os.WriteFile(f1, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Change byte after partial hash boundary.
	data[4096] = 0xFF
	if err := os.WriteFile(f2, data, 0o644); err != nil {
		t.Fatal(err)
	}

	h1, err := hashFilePartial(f1)
	if err != nil {
		t.Fatalf("hashFilePartial(%s): %v", f1, err)
	}

	h2, err := hashFilePartial(f2)
	if err != nil {
		t.Fatalf("hashFilePartial(%s): %v", f2, err)
	}

	// Partial hashes should be the same (only reads first 4KB).
	if h1 != h2 {
		t.Errorf("partial hashes differ for files with identical first 4KB: %s vs %s", h1, h2)
	}
}

func TestHashFilePartial_DifferentPrefix(t *testing.T) {
	dir := t.TempDir()

	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")

	data1 := make([]byte, 4096)
	data2 := make([]byte, 4096)
	for i := range data1 {
		data1[i] = byte(i % 256)
		data2[i] = byte((i + 1) % 256)
	}

	if err := os.WriteFile(f1, data1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, data2, 0o644); err != nil {
		t.Fatal(err)
	}

	h1, err := hashFilePartial(f1)
	if err != nil {
		t.Fatal(err)
	}

	h2, err := hashFilePartial(f2)
	if err != nil {
		t.Fatal(err)
	}

	if h1 == h2 {
		t.Error("partial hashes should differ for files with different first 4KB")
	}
}

func TestHashFilePartial_SmallFile(t *testing.T) {
	dir := t.TempDir()

	// File smaller than partialHashSize.
	f := filepath.Join(dir, "tiny")
	if err := os.WriteFile(f, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := hashFilePartial(f)
	if err != nil {
		t.Fatalf("hashFilePartial: %v", err)
	}
	if h == "" {
		t.Error("expected non-empty hash for small file")
	}
}

func TestHashFileFull_IdenticalFiles(t *testing.T) {
	dir := t.TempDir()

	data := make([]byte, 16384)
	for i := range data {
		data[i] = byte(i % 256)
	}

	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")

	if err := os.WriteFile(f1, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, data, 0o644); err != nil {
		t.Fatal(err)
	}

	h1, err := hashFileFull(f1, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	h2, err := hashFileFull(f2, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	if h1 != h2 {
		t.Errorf("full hashes differ for identical files: %s vs %s", h1, h2)
	}
}

func TestHashFileFull_DifferentFiles(t *testing.T) {
	dir := t.TempDir()

	f1 := filepath.Join(dir, "f1")
	f2 := filepath.Join(dir, "f2")

	data1 := make([]byte, 16384)
	data2 := make([]byte, 16384)
	for i := range data1 {
		data1[i] = byte(i % 256)
		data2[i] = byte((i + 7) % 256)
	}

	if err := os.WriteFile(f1, data1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, data2, 0o644); err != nil {
		t.Fatal(err)
	}

	h1, err := hashFileFull(f1, int64(len(data1)))
	if err != nil {
		t.Fatal(err)
	}

	h2, err := hashFileFull(f2, int64(len(data2)))
	if err != nil {
		t.Fatal(err)
	}

	if h1 == h2 {
		t.Error("full hashes should differ for different files")
	}
}

func TestHashFileFull_SizeChanged(t *testing.T) {
	dir := t.TempDir()

	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pass wrong expected size — should return error.
	_, err := hashFileFull(f, 999)
	if err == nil {
		t.Error("expected error when file size changed")
	}
}

func TestHashFileFull_LargeFile(t *testing.T) {
	dir := t.TempDir()

	// 1MB file to test streaming.
	size := 1 << 20
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251) // prime to avoid patterns
	}

	f := filepath.Join(dir, "large")
	if err := os.WriteFile(f, data, 0o644); err != nil {
		t.Fatal(err)
	}

	h, err := hashFileFull(f, int64(size))
	if err != nil {
		t.Fatalf("hashFileFull: %v", err)
	}
	if h == "" {
		t.Error("expected non-empty hash")
	}

	// Hash again — should be deterministic.
	h2, err := hashFileFull(f, int64(size))
	if err != nil {
		t.Fatal(err)
	}
	if h != h2 {
		t.Errorf("hash not deterministic: %s vs %s", h, h2)
	}
}
