//go:build darwin

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// runConserveMode moves duplicate files to the conserve directory, keeping the
// first file in each group as the keeper.
func runConserveMode(groups []DupeGroup, conserveDir, sourceDir string) error {
	if len(groups) == 0 {
		fmt.Println("No duplicates to conserve.")
		return nil
	}

	absConserve, err := filepath.Abs(conserveDir)
	if err != nil {
		return fmt.Errorf("resolving conserve dir: %w", err)
	}

	if err := os.MkdirAll(absConserve, 0o755); err != nil {
		return fmt.Errorf("creating conserve dir: %w", err)
	}

	manifest, err := loadManifest(absConserve)
	if err != nil {
		manifest = &Manifest{
			Version:   1,
			Created:   time.Now(),
			SourceDir: sourceDir,
			Entries:   []ManifestEntry{},
		}
	}

	var totalConserved int
	var totalBytes int64

	for _, g := range groups {
		// First file is the keeper.
		for _, f := range g.Files[1:] {
			if err := conserveFile(f.Path, absConserve, g.Hash, manifest); err != nil {
				fmt.Fprintf(os.Stderr, "bw dupes: conserve %s: %v\n", displayPath(f.Path), err)
				continue
			}

			totalConserved++
			totalBytes += f.Size
			fmt.Printf("  %s→%s conserved %s\n", colorCyan, colorReset, displayPath(f.Path))

			// Write manifest after each move for crash safety.
			if err := writeManifest(absConserve, manifest); err != nil {
				return fmt.Errorf("writing manifest: %w", err)
			}
		}
	}

	fmt.Printf("\n%sConserved %d files%s (%s) to %s\n",
		colorGreen, totalConserved, colorReset, humanizeBytes(totalBytes), displayPath(absConserve))
	fmt.Printf("Restore with: bw dupes --restore %s\n", displayPath(absConserve))

	return nil
}

// conserveFile moves a file to the conserve directory, preserving its absolute
// path structure. Uses copy+verify+delete for cross-volume moves.
func conserveFile(srcPath, conserveDir, hash string, manifest *Manifest) error {
	absSrc, err := filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("resolving source: %w", err)
	}

	// Preserve full path structure inside conserve dir.
	conservedPath := filepath.Join(conserveDir, absSrc)

	if err := os.MkdirAll(filepath.Dir(conservedPath), 0o755); err != nil {
		return fmt.Errorf("creating conserve subdirs: %w", err)
	}

	info, err := os.Stat(absSrc)
	if err != nil {
		return err
	}

	if isCrossDevice(absSrc, conserveDir) {
		// Cross-volume: copy → verify hash → delete original.
		if err := copyFile(absSrc, conservedPath); err != nil {
			return fmt.Errorf("cross-volume copy: %w", err)
		}

		// Verify the copy.
		copyHash, err := hashFileFull(conservedPath, info.Size())
		if err != nil {
			os.Remove(conservedPath)
			return fmt.Errorf("verify copy hash: %w", err)
		}
		if copyHash != hash {
			os.Remove(conservedPath)
			return fmt.Errorf("copy hash mismatch: got %s, want %s", copyHash, hash)
		}

		if err := os.Remove(absSrc); err != nil {
			return fmt.Errorf("remove original after copy: %w", err)
		}
	} else {
		// Same volume: atomic rename.
		if err := os.Rename(absSrc, conservedPath); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	}

	manifest.Entries = append(manifest.Entries, ManifestEntry{
		OriginalPath:  absSrc,
		ConservedPath: conservedPath,
		Hash:          hash,
		Size:          info.Size(),
		MovedAt:       time.Now(),
	})

	return nil
}

// runRestoreMode reads the manifest and restores files to their original locations.
// If filterPath is non-empty, only that single file is restored.
func runRestoreMode(conserveDir, filterPath string) error {
	absConserve, err := filepath.Abs(conserveDir)
	if err != nil {
		return fmt.Errorf("resolving conserve dir: %w", err)
	}

	manifest, err := loadManifest(absConserve)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if len(manifest.Entries) == 0 {
		fmt.Println("No files to restore.")
		return nil
	}

	var remaining []ManifestEntry
	var restored int

	for _, entry := range manifest.Entries {
		if filterPath != "" && entry.OriginalPath != filterPath {
			remaining = append(remaining, entry)
			continue
		}

		if err := restoreFile(entry); err != nil {
			fmt.Fprintf(os.Stderr, "bw dupes: restore %s: %v\n", displayPath(entry.OriginalPath), err)
			remaining = append(remaining, entry)
			continue
		}

		restored++
		fmt.Printf("  %s✓%s restored %s\n", colorGreen, colorReset, displayPath(entry.OriginalPath))
	}

	// Update manifest with remaining entries.
	manifest.Entries = remaining
	if len(remaining) == 0 {
		// All restored — remove manifest.
		os.Remove(filepath.Join(absConserve, manifestFilename))
	} else {
		if err := writeManifest(absConserve, manifest); err != nil {
			return fmt.Errorf("updating manifest: %w", err)
		}
	}

	fmt.Printf("\n%sRestored %d files.%s\n", colorGreen, restored, colorReset)

	return nil
}

// restoreFile moves a single conserved file back to its original location.
func restoreFile(entry ManifestEntry) error {
	// Check conserved file exists.
	info, err := os.Stat(entry.ConservedPath)
	if err != nil {
		return fmt.Errorf("conserved file missing: %w", err)
	}

	// Verify hash before restoring (detect corruption).
	currentHash, err := hashFileFull(entry.ConservedPath, info.Size())
	if err != nil {
		return fmt.Errorf("hash check: %w", err)
	}
	if currentHash != entry.Hash {
		return fmt.Errorf("hash mismatch for %s: expected %s, got %s (possible corruption)",
			entry.ConservedPath, entry.Hash, currentHash)
	}

	// Check original location is free.
	if _, err := os.Stat(entry.OriginalPath); err == nil {
		return fmt.Errorf("original path already exists: %s", entry.OriginalPath)
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(entry.OriginalPath), 0o755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}

	if isCrossDevice(entry.ConservedPath, filepath.Dir(entry.OriginalPath)) {
		if err := copyFile(entry.ConservedPath, entry.OriginalPath); err != nil {
			return fmt.Errorf("cross-volume copy: %w", err)
		}
		if err := os.Remove(entry.ConservedPath); err != nil {
			return fmt.Errorf("remove conserved after restore: %w", err)
		}
	} else {
		if err := os.Rename(entry.ConservedPath, entry.OriginalPath); err != nil {
			return fmt.Errorf("rename: %w", err)
		}
	}

	return nil
}

// writeManifest writes the manifest to the conserve directory.
func writeManifest(conserveDir string, manifest *Manifest) error {
	path := filepath.Join(conserveDir, manifestFilename)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	// Write atomically via temp file.
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp manifest: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming manifest: %w", err)
	}

	return nil
}

// loadManifest reads the manifest from the conserve directory.
func loadManifest(conserveDir string) (*Manifest, error) {
	path := filepath.Join(conserveDir, manifestFilename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}

// isCrossDevice checks if two paths are on different filesystems.
func isCrossDevice(path1, path2 string) bool {
	var stat1, stat2 syscall.Stat_t

	if err := syscall.Stat(path1, &stat1); err != nil {
		return true // assume cross-device on error (safer)
	}
	if err := syscall.Stat(path2, &stat2); err != nil {
		return true
	}

	return stat1.Dev != stat2.Dev
}

// copyFile copies src to dst, preserving permissions.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)

	if _, err := io.CopyBuffer(dstFile, srcFile, *bufPtr); err != nil {
		os.Remove(dst)
		return err
	}

	return dstFile.Close()
}
