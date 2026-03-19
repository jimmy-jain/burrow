//go:build darwin

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDuplicates_NoDupes(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a.txt"), 1024)
	writeFileContent(t, filepath.Join(dir, "b.txt"), 1024, 1) // different content

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatalf("findDuplicates: %v", err)
	}

	if len(groups) != 0 {
		t.Errorf("expected 0 dupe groups, got %d", len(groups))
	}
}

func TestFindDuplicates_TwoIdenticalFiles(t *testing.T) {
	dir := t.TempDir()

	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatalf("findDuplicates: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 dupe group, got %d", len(groups))
	}
	if len(groups[0].Files) != 2 {
		t.Errorf("expected 2 files in group, got %d", len(groups[0].Files))
	}
	if groups[0].Hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestFindDuplicates_ThreeCopies(t *testing.T) {
	dir := t.TempDir()

	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 251)
	}

	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Files) != 3 {
		t.Errorf("expected 3 files in group, got %d", len(groups[0].Files))
	}
	if groups[0].ReclaimableBytes() != int64(len(data))*2 {
		t.Errorf("expected %d reclaimable, got %d", int64(len(data))*2, groups[0].ReclaimableBytes())
	}
}

func TestFindDuplicates_MultipleGroups(t *testing.T) {
	dir := t.TempDir()

	// Group 1: two copies of content A (4KB).
	dataA := make([]byte, 4096)
	for i := range dataA {
		dataA[i] = byte(i % 256)
	}
	os.WriteFile(filepath.Join(dir, "a1.txt"), dataA, 0o644)
	os.WriteFile(filepath.Join(dir, "a2.txt"), dataA, 0o644)

	// Group 2: two copies of content B (8KB).
	dataB := make([]byte, 8192)
	for i := range dataB {
		dataB[i] = byte((i + 42) % 256)
	}
	os.WriteFile(filepath.Join(dir, "b1.txt"), dataB, 0o644)
	os.WriteFile(filepath.Join(dir, "b2.txt"), dataB, 0o644)

	// Unique file — no dupe.
	writeFileContent(t, filepath.Join(dir, "unique.txt"), 4096, 99)

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 2 {
		t.Fatalf("expected 2 dupe groups, got %d", len(groups))
	}

	// Groups should be sorted by reclaimable space descending.
	if groups[0].ReclaimableBytes() < groups[1].ReclaimableBytes() {
		t.Errorf("groups not sorted by reclaimable descending: %d < %d",
			groups[0].ReclaimableBytes(), groups[1].ReclaimableBytes())
	}
}

func TestFindDuplicates_SameSizeDifferentContent(t *testing.T) {
	dir := t.TempDir()

	// Two files with same size but different content.
	writeFileContent(t, filepath.Join(dir, "x.txt"), 2048, 0)
	writeFileContent(t, filepath.Join(dir, "y.txt"), 2048, 1)

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for same-size different-content, got %d", len(groups))
	}
}

func TestFindDuplicates_Subdirectories(t *testing.T) {
	dir := t.TempDir()

	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 200)
	}

	os.MkdirAll(filepath.Join(dir, "sub1"), 0o755)
	os.MkdirAll(filepath.Join(dir, "sub2", "deep"), 0o755)
	os.WriteFile(filepath.Join(dir, "sub1", "f.txt"), data, 0o644)
	os.WriteFile(filepath.Join(dir, "sub2", "deep", "f.txt"), data, 0o644)

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 dupe group across subdirs, got %d", len(groups))
	}
}

func TestFindDuplicates_MinSizeFilter(t *testing.T) {
	dir := t.TempDir()

	// Small dupes (below threshold).
	small := make([]byte, 512)
	os.WriteFile(filepath.Join(dir, "s1.txt"), small, 0o644)
	os.WriteFile(filepath.Join(dir, "s2.txt"), small, 0o644)

	// Large dupes (above threshold).
	large := make([]byte, 2048)
	for i := range large {
		large[i] = byte(i % 256)
	}
	os.WriteFile(filepath.Join(dir, "l1.txt"), large, 0o644)
	os.WriteFile(filepath.Join(dir, "l2.txt"), large, 0o644)

	cfg := ScanConfig{Root: dir, MinSize: 1024}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group (small dupes filtered), got %d", len(groups))
	}
}

func TestFindDuplicates_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty dir, got %d", len(groups))
	}
}

func TestFindDuplicates_HardlinksExcluded(t *testing.T) {
	dir := t.TempDir()

	original := filepath.Join(dir, "original.txt")
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if err := os.WriteFile(original, data, 0o644); err != nil {
		t.Fatal(err)
	}

	hardlink := filepath.Join(dir, "hardlink.txt")
	if err := os.Link(original, hardlink); err != nil {
		t.Skipf("hardlinks not supported: %v", err)
	}

	cfg := ScanConfig{Root: dir, MinSize: 0}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) != 0 {
		t.Errorf("hardlinks should not be reported as duplicates, got %d groups", len(groups))
	}
}

// writeFileContent creates a file with deterministic content seeded by offset.
func writeFileContent(t *testing.T, path string, size int, offset int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i + offset) % 256)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
