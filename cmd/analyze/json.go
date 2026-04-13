//go:build darwin

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync/atomic"
	"time"
)

type jsonOutput struct {
	Path       string          `json:"path"`
	Overview   bool            `json:"overview"`
	Entries    []jsonEntry     `json:"entries"`
	LargeFiles []jsonFileEntry `json:"large_files,omitempty"`
	TotalSize  int64           `json:"total_size"`
	TotalFiles int64           `json:"total_files,omitempty"`
}

type jsonEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	IsDir      bool   `json:"is_dir"`
	Cleanable  bool   `json:"cleanable,omitempty"`
	LastAccess string `json:"last_access,omitempty"`
}

type jsonFileEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func runJSONMode(path string, isOverview bool) {
	result := performScanForJSON(path, isOverview)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}

func performScanForJSON(path string, isOverview bool) jsonOutput {
	if isOverview {
		return performOverviewScanForJSON(path)
	}
	return performDirectoryScanForJSON(path)
}

func performDirectoryScanForJSON(path string) jsonOutput {
	var filesScanned, dirsScanned, bytesScanned int64
	currentPath := &atomic.Value{}
	currentPath.Store("")

	result, err := scanPathConcurrent(path, &filesScanned, &dirsScanned, &bytesScanned, currentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to scan directory: %v\n", err)
		os.Exit(1)
	}

	return jsonOutput{
		Path:       path,
		Overview:   false,
		Entries:    jsonEntriesFromDirEntries(result.Entries),
		LargeFiles: jsonFileEntriesFromFileEntries(result.LargeFiles),
		TotalSize:  result.TotalSize,
		TotalFiles: result.TotalFiles,
	}
}

func performOverviewScanForJSON(path string) jsonOutput {
	overviewEntries := createOverviewEntries()

	var totalSize int64
	entries := make([]dirEntry, 0, len(overviewEntries))
	for _, entry := range overviewEntries {
		var size int64

		if cached, cacheErr := loadOverviewCachedSize(entry.Path); cacheErr == nil && cached > 0 {
			size = cached
		} else {
			var err error
			size, err = measureOverviewSize(entry.Path)
			if err != nil {
				continue
			}
		}

		if size == 0 {
			continue
		}
		entry.Size = size
		totalSize += size
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})

	return jsonOutput{
		Path:      path,
		Overview:  true,
		Entries:   jsonEntriesFromDirEntries(entries),
		TotalSize: totalSize,
	}
}

func jsonEntriesFromDirEntries(entries []dirEntry) []jsonEntry {
	output := make([]jsonEntry, 0, len(entries))
	for _, entry := range entries {
		item := jsonEntry{
			Name:      entry.Name,
			Path:      entry.Path,
			Size:      entry.Size,
			IsDir:     entry.IsDir,
			Cleanable: entry.IsDir && isCleanableDir(entry.Path),
		}

		if !entry.LastAccess.IsZero() {
			item.LastAccess = entry.LastAccess.UTC().Format(time.RFC3339)
		}

		output = append(output, item)
	}
	return output
}

func jsonFileEntriesFromFileEntries(files []fileEntry) []jsonFileEntry {
	output := make([]jsonFileEntry, 0, len(files))
	for _, f := range files {
		output = append(output, jsonFileEntry(f))
	}
	return output
}
