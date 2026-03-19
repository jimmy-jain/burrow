//go:build darwin

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFiles_BasicWalk(t *testing.T) {
	dir := t.TempDir()

	// Create test files of varying sizes.
	writeFile(t, filepath.Join(dir, "a.txt"), 1024)
	writeFile(t, filepath.Join(dir, "b.txt"), 2048)
	writeFile(t, filepath.Join(dir, "sub", "c.txt"), 4096)

	cfg := ScanConfig{Root: dir, MinSize: 0}
	files, err := scanFiles(cfg)
	if err != nil {
		t.Fatalf("scanFiles: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
}

func TestScanFiles_SkipsSmallFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "small.txt"), 100)
	writeFile(t, filepath.Join(dir, "big.txt"), 2048)

	cfg := ScanConfig{Root: dir, MinSize: 1024}
	files, err := scanFiles(cfg)
	if err != nil {
		t.Fatalf("scanFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file (big only), got %d", len(files))
	}
	if files[0].Size != 2048 {
		t.Errorf("expected size 2048, got %d", files[0].Size)
	}
}

func TestScanFiles_SkipsSymlinks(t *testing.T) {
	dir := t.TempDir()

	realFile := filepath.Join(dir, "real.txt")
	writeFile(t, realFile, 1024)

	linkPath := filepath.Join(dir, "link.txt")
	if err := os.Symlink(realFile, linkPath); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	cfg := ScanConfig{Root: dir, MinSize: 0}
	files, err := scanFiles(cfg)
	if err != nil {
		t.Fatalf("scanFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file (symlink skipped), got %d", len(files))
	}
}

func TestScanFiles_SkipsDirs(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "keep.txt"), 1024)
	writeFile(t, filepath.Join(dir, ".git", "objects", "pack"), 4096)
	writeFile(t, filepath.Join(dir, "node_modules", "pkg", "index.js"), 2048)

	cfg := ScanConfig{Root: dir, MinSize: 0}
	files, err := scanFiles(cfg)
	if err != nil {
		t.Fatalf("scanFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file (skipped dirs excluded), got %d", len(files))
	}
}

func TestScanFiles_ExcludesConserveDir(t *testing.T) {
	dir := t.TempDir()
	conserveDir := filepath.Join(dir, "conserve")

	writeFile(t, filepath.Join(dir, "real.txt"), 1024)
	writeFile(t, filepath.Join(conserveDir, "old.txt"), 2048)

	cfg := ScanConfig{Root: dir, MinSize: 0, ConserveDir: conserveDir}
	files, err := scanFiles(cfg)
	if err != nil {
		t.Fatalf("scanFiles: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file (conserve dir excluded), got %d", len(files))
	}
}

func TestScanFiles_PermissionError(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "ok.txt"), 1024)

	noAccessDir := filepath.Join(dir, "noaccess")
	if err := os.MkdirAll(noAccessDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(noAccessDir, "secret.txt"), 2048)
	if err := os.Chmod(noAccessDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(noAccessDir, 0o755) })

	cfg := ScanConfig{Root: dir, MinSize: 0}
	files, err := scanFiles(cfg)
	if err != nil {
		t.Fatalf("scanFiles should not fail on permission error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 accessible file, got %d", len(files))
	}
}

func TestGroupBySize(t *testing.T) {
	files := []FileEntry{
		{Path: "/a", Size: 100, Inode: 1, Device: 1},
		{Path: "/b", Size: 200, Inode: 2, Device: 1},
		{Path: "/c", Size: 100, Inode: 3, Device: 1},
		{Path: "/d", Size: 200, Inode: 4, Device: 1},
		{Path: "/e", Size: 300, Inode: 5, Device: 1},
	}

	groups := groupBySize(files)

	// Only sizes with 2+ files should remain.
	if len(groups) != 2 {
		t.Fatalf("expected 2 size groups, got %d", len(groups))
	}

	for size, group := range groups {
		if len(group) != 2 {
			t.Errorf("size %d: expected 2 files, got %d", size, len(group))
		}
	}
}

func TestDeduplicateByInode(t *testing.T) {
	files := []FileEntry{
		{Path: "/a", Size: 100, Inode: 1, Device: 1},
		{Path: "/b", Size: 100, Inode: 1, Device: 1}, // hardlink of /a
		{Path: "/c", Size: 100, Inode: 2, Device: 1},
	}

	deduped := deduplicateByInode(files)

	if len(deduped) != 2 {
		t.Fatalf("expected 2 unique inodes, got %d", len(deduped))
	}
}

func TestDeduplicateByInode_DifferentDevices(t *testing.T) {
	files := []FileEntry{
		{Path: "/a", Size: 100, Inode: 1, Device: 1},
		{Path: "/b", Size: 100, Inode: 1, Device: 2}, // same inode, different device = different file
	}

	deduped := deduplicateByInode(files)

	if len(deduped) != 2 {
		t.Fatalf("expected 2 files (different devices), got %d", len(deduped))
	}
}

// writeFile creates a file with the given size filled with deterministic data.
func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
