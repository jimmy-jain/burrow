//go:build darwin

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

// scanFiles walks root and returns all regular files matching the config filters.
// Skips symlinks, skip-listed directories, and files below MinSize.
func scanFiles(cfg ScanConfig) ([]FileEntry, error) {
	root := filepath.Clean(cfg.Root)

	// Resolve symlinks on the root path itself (e.g., /tmp -> /private/tmp).
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", root, err)
	}
	root = resolved

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}

	isRootDir := root == "/"

	// Resolve conserve dir to absolute for comparison.
	var conserveAbs string
	if cfg.ConserveDir != "" {
		abs, _ := filepath.Abs(cfg.ConserveDir)
		// Resolve symlinks so the comparison works on macOS where
		// /var -> /private/var.
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			conserveAbs = resolved
		} else {
			conserveAbs = abs
		}
	}

	numWorkers := max(min(runtime.NumCPU()*scanCPUMult, maxScanWorkers), minScanWorkers)
	sem := make(chan struct{}, numWorkers)

	var mu sync.Mutex
	var files []FileEntry
	var wg sync.WaitGroup
	var filesScanned int64

	var walk func(string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			// Permission error or similar — skip silently.
			return
		}

		for _, entry := range entries {
			name := entry.Name()
			fullPath := filepath.Join(dir, name)

			// Skip symlinks entirely.
			if entry.Type()&fs.ModeSymlink != 0 {
				continue
			}

			if entry.IsDir() {
				// Skip well-known dirs.
				if skipDirs[name] {
					continue
				}
				// Skip system root dirs when scanning /.
				if isRootDir && skipSystemRootDirs[name] {
					continue
				}
				// Skip conserve dir if inside scan root.
				if conserveAbs != "" && fullPath == conserveAbs {
					continue
				}

				select {
				case sem <- struct{}{}:
					wg.Add(1)
					go func(p string) {
						defer wg.Done()
						defer func() { <-sem }()
						walk(p)
					}(fullPath)
				default:
					// Fallback to synchronous to avoid deadlock.
					walk(fullPath)
				}
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			size := info.Size()
			if size < cfg.MinSize {
				continue
			}

			// Extract inode and device from syscall stat.
			var inode, device uint64
			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				inode = stat.Ino
				device = uint64(stat.Dev)
			}

			fe := FileEntry{
				Path:   fullPath,
				Size:   size,
				Inode:  inode,
				Device: device,
			}

			mu.Lock()
			files = append(files, fe)
			mu.Unlock()
			atomic.AddInt64(&filesScanned, 1)
		}
	}

	walk(root)
	wg.Wait()

	return files, nil
}

// groupBySize partitions files by size, discarding groups with only one file.
func groupBySize(files []FileEntry) map[int64][]FileEntry {
	sizeMap := make(map[int64][]FileEntry, len(files)/2)
	for _, f := range files {
		sizeMap[f.Size] = append(sizeMap[f.Size], f)
	}

	// Remove singletons — unique sizes can't be duplicates.
	for size, group := range sizeMap {
		if len(group) < 2 {
			delete(sizeMap, size)
		}
	}

	return sizeMap
}

// deduplicateByInode removes files that share the same inode+device
// (hardlinks), keeping only one representative per unique inode.
func deduplicateByInode(files []FileEntry) []FileEntry {
	type inodeKey struct {
		inode  uint64
		device uint64
	}

	seen := make(map[inodeKey]bool, len(files))
	result := make([]FileEntry, 0, len(files))

	for _, f := range files {
		if f.Inode == 0 {
			// No inode info available — keep the file.
			result = append(result, f)
			continue
		}
		key := inodeKey{f.Inode, f.Device}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, f)
	}

	return result
}

// displayPath replaces the home directory prefix with ~.
func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
