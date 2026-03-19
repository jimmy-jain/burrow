//go:build darwin

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConserveFile_MovesAndPreservesPath(t *testing.T) {
	srcDir := t.TempDir()
	conserveDir := t.TempDir()

	// Create source file.
	srcFile := filepath.Join(srcDir, "sub", "report.pdf")
	writeFile(t, srcFile, 2048)

	manifest := &Manifest{Version: 1, SourceDir: srcDir}

	err := conserveFile(srcFile, conserveDir, "abc123", manifest)
	if err != nil {
		t.Fatalf("conserveFile: %v", err)
	}

	// Original should be gone.
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("original file should be removed after conserve")
	}

	// Conserved file should exist at path-preserving location.
	expectedConserved := filepath.Join(conserveDir, srcFile)
	if _, err := os.Stat(expectedConserved); err != nil {
		t.Errorf("conserved file not found at %s: %v", expectedConserved, err)
	}

	// Manifest should have one entry.
	if len(manifest.Entries) != 1 {
		t.Fatalf("expected 1 manifest entry, got %d", len(manifest.Entries))
	}
	entry := manifest.Entries[0]
	if entry.OriginalPath != srcFile {
		t.Errorf("original_path = %s, want %s", entry.OriginalPath, srcFile)
	}
	if entry.Hash != "abc123" {
		t.Errorf("hash = %s, want abc123", entry.Hash)
	}
}

func TestConserveFile_VerifiesHash(t *testing.T) {
	srcDir := t.TempDir()
	conserveDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "data.bin")
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(srcFile, data, 0o644)

	// Compute real hash.
	hash, err := hashFileFull(srcFile, 2048)
	if err != nil {
		t.Fatal(err)
	}

	manifest := &Manifest{Version: 1, SourceDir: srcDir}
	err = conserveFile(srcFile, conserveDir, hash, manifest)
	if err != nil {
		t.Fatalf("conserveFile: %v", err)
	}

	// Verify the conserved file hash matches.
	conservedPath := manifest.Entries[0].ConservedPath
	conservedHash, err := hashFileFull(conservedPath, 2048)
	if err != nil {
		t.Fatalf("hash conserved file: %v", err)
	}
	if conservedHash != hash {
		t.Errorf("conserved hash %s != original hash %s", conservedHash, hash)
	}
}

func TestWriteManifest_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	manifest := &Manifest{
		Version:   1,
		SourceDir: "/tmp/src",
		Entries: []ManifestEntry{
			{
				OriginalPath:  "/tmp/src/a.txt",
				ConservedPath: "/tmp/conserve/tmp/src/a.txt",
				Hash:          "abc123",
				Size:          1024,
			},
		},
	}

	if err := writeManifest(dir, manifest); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	// Read it back.
	loaded, err := loadManifest(dir)
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("version = %d, want 1", loaded.Version)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Hash != "abc123" {
		t.Errorf("hash = %s, want abc123", loaded.Entries[0].Hash)
	}
}

func TestLoadManifest_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := loadManifest(dir)
	if err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestRestoreFile_Basic(t *testing.T) {
	conserveDir := t.TempDir()
	originalDir := t.TempDir()

	// Simulate a conserved file.
	originalPath := filepath.Join(originalDir, "doc.txt")
	conservedPath := filepath.Join(conserveDir, originalPath)
	writeFile(t, conservedPath, 2048)

	hash, err := hashFileFull(conservedPath, 2048)
	if err != nil {
		t.Fatal(err)
	}

	entry := ManifestEntry{
		OriginalPath:  originalPath,
		ConservedPath: conservedPath,
		Hash:          hash,
		Size:          2048,
	}

	err = restoreFile(entry)
	if err != nil {
		t.Fatalf("restoreFile: %v", err)
	}

	// Original should be restored.
	if _, err := os.Stat(originalPath); err != nil {
		t.Errorf("original not restored: %v", err)
	}

	// Conserved copy should be removed.
	if _, err := os.Stat(conservedPath); !os.IsNotExist(err) {
		t.Error("conserved file should be removed after restore")
	}
}

func TestRestoreFile_HashMismatch(t *testing.T) {
	conserveDir := t.TempDir()

	conservedPath := filepath.Join(conserveDir, "corrupt.txt")
	writeFile(t, conservedPath, 2048)

	entry := ManifestEntry{
		OriginalPath:  "/tmp/doesnotmatter",
		ConservedPath: conservedPath,
		Hash:          "wrong_hash_value_",
		Size:          2048,
	}

	err := restoreFile(entry)
	if err == nil {
		t.Error("expected error for hash mismatch")
	}
}

func TestRestoreFile_OriginalAlreadyExists(t *testing.T) {
	conserveDir := t.TempDir()
	originalDir := t.TempDir()

	originalPath := filepath.Join(originalDir, "existing.txt")
	conservedPath := filepath.Join(conserveDir, originalPath)

	// Both original and conserved exist.
	writeFile(t, originalPath, 1024)
	writeFile(t, conservedPath, 2048)

	entry := ManifestEntry{
		OriginalPath:  originalPath,
		ConservedPath: conservedPath,
		Hash:          "abc",
		Size:          2048,
	}

	err := restoreFile(entry)
	if err == nil {
		t.Error("expected error when original already exists")
	}
}

func TestRunConserveMode_EndToEnd(t *testing.T) {
	srcDir := t.TempDir()
	conserveDir := t.TempDir()

	// Create duplicate files.
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(filepath.Join(srcDir, "a.txt"), data, 0o644)
	os.WriteFile(filepath.Join(srcDir, "b.txt"), data, 0o644)

	cfg := ScanConfig{Root: srcDir, MinSize: 0, ConserveDir: conserveDir}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) == 0 {
		t.Fatal("expected duplicates")
	}

	err = runConserveMode(groups, conserveDir, srcDir)
	if err != nil {
		t.Fatalf("runConserveMode: %v", err)
	}

	// Keeper should still exist.
	if _, err := os.Stat(filepath.Join(srcDir, "a.txt")); err != nil {
		t.Error("keeper file should still exist")
	}

	// Dupe should be moved.
	if _, err := os.Stat(filepath.Join(srcDir, "b.txt")); !os.IsNotExist(err) {
		t.Error("duplicate should be moved to conserve dir")
	}

	// Manifest should exist.
	manifest, err := loadManifest(conserveDir)
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}
	if len(manifest.Entries) != 1 {
		t.Errorf("expected 1 manifest entry, got %d", len(manifest.Entries))
	}
}

func TestRunRestoreMode_EndToEnd(t *testing.T) {
	srcDir := t.TempDir()
	conserveDir := t.TempDir()

	// Create and conserve duplicates.
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(filepath.Join(srcDir, "a.txt"), data, 0o644)
	os.WriteFile(filepath.Join(srcDir, "b.txt"), data, 0o644)

	cfg := ScanConfig{Root: srcDir, MinSize: 0, ConserveDir: conserveDir}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = runConserveMode(groups, conserveDir, srcDir)
	if err != nil {
		t.Fatal(err)
	}

	// Now restore.
	err = runRestoreMode(conserveDir, "")
	if err != nil {
		t.Fatalf("runRestoreMode: %v", err)
	}

	// Both files should exist again.
	for _, name := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(srcDir, name)); err != nil {
			t.Errorf("file %s should be restored: %v", name, err)
		}
	}
}

func TestRunRestoreMode_SingleFile(t *testing.T) {
	srcDir := t.TempDir()
	conserveDir := t.TempDir()

	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 256)
	}
	f1 := filepath.Join(srcDir, "a.txt")
	f2 := filepath.Join(srcDir, "b.txt")
	f3 := filepath.Join(srcDir, "c.txt")
	os.WriteFile(f1, data, 0o644)
	os.WriteFile(f2, data, 0o644)
	os.WriteFile(f3, data, 0o644)

	cfg := ScanConfig{Root: srcDir, MinSize: 0, ConserveDir: conserveDir}
	groups, err := findDuplicates(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = runConserveMode(groups, conserveDir, srcDir)
	if err != nil {
		t.Fatal(err)
	}

	// Restore only one specific file.
	manifest, _ := loadManifest(conserveDir)
	if len(manifest.Entries) < 1 {
		t.Fatal("need at least 1 conserved entry")
	}
	targetOriginal := manifest.Entries[0].OriginalPath

	err = runRestoreMode(conserveDir, targetOriginal)
	if err != nil {
		t.Fatalf("runRestoreMode single file: %v", err)
	}

	// Target should be restored.
	if _, err := os.Stat(targetOriginal); err != nil {
		t.Errorf("target file should be restored: %v", err)
	}

	// Manifest should still have remaining entries.
	updatedManifest, _ := loadManifest(conserveDir)
	if len(updatedManifest.Entries) != len(manifest.Entries)-1 {
		t.Errorf("manifest should have %d entries after single restore, got %d",
			len(manifest.Entries)-1, len(updatedManifest.Entries))
	}
}

func TestManifest_JSONRoundtrip(t *testing.T) {
	original := Manifest{
		Version:   1,
		SourceDir: "/Users/jimmy/Documents",
		Entries: []ManifestEntry{
			{
				OriginalPath:  "/Users/jimmy/Documents/report.pdf",
				ConservedPath: "/tmp/conserve/Users/jimmy/Documents/report.pdf",
				Hash:          "0123456789abcdef",
				Size:          15900000,
			},
		},
	}

	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var loaded Manifest
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.Version != original.Version {
		t.Errorf("version mismatch: %d != %d", loaded.Version, original.Version)
	}
	if loaded.SourceDir != original.SourceDir {
		t.Errorf("source_dir mismatch")
	}
	if len(loaded.Entries) != 1 {
		t.Fatal("entry count mismatch")
	}
	if loaded.Entries[0].Hash != original.Entries[0].Hash {
		t.Error("hash mismatch")
	}
}
