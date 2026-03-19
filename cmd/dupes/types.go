//go:build darwin

package main

import "time"

// FileEntry represents a file discovered during scanning.
type FileEntry struct {
	Path   string
	Size   int64
	Inode  uint64
	Device uint64
}

// DupeGroup is a set of files with identical content.
type DupeGroup struct {
	Hash  string
	Size  int64
	Files []FileEntry
}

// ReclaimableBytes returns total bytes reclaimable by removing all but one copy.
func (g DupeGroup) ReclaimableBytes() int64 {
	if len(g.Files) <= 1 {
		return 0
	}
	return g.Size * int64(len(g.Files)-1)
}

// Manifest records all files moved during a conserve operation.
type Manifest struct {
	Version   int             `json:"version"`
	Created   time.Time       `json:"created"`
	SourceDir string          `json:"source_dir"`
	Entries   []ManifestEntry `json:"entries"`
}

// ManifestEntry records a single conserved file.
type ManifestEntry struct {
	OriginalPath  string    `json:"original_path"`
	ConservedPath string    `json:"conserved_path"`
	Hash          string    `json:"hash"`
	Size          int64     `json:"size"`
	MovedAt       time.Time `json:"moved_at"`
}

// ScanProgress tracks scanning stats for progress display.
type ScanProgress struct {
	FilesScanned int64
	DirsScanned  int64
	BytesScanned int64
}

// ScanConfig controls scanning behavior.
type ScanConfig struct {
	Root       string
	MinSize    int64
	ConserveDir string // excluded from scan when inside root
}
