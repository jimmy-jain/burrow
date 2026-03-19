//go:build darwin

package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
)

// findDuplicates orchestrates the 6-phase duplicate detection pipeline:
// 1. Walk — collect files
// 2. Group by size — different sizes can't be duplicates
// 3. Deduplicate by inode — hardlinks are the same file
// 4. Partial hash — fast rejection of same-size different files
// 5. Full hash — confirm true duplicates
// 6. Sort — by reclaimable space descending
func findDuplicates(cfg ScanConfig) ([]DupeGroup, error) {
	// Phase 1: Walk.
	files, err := scanFiles(cfg)
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", cfg.Root, err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	// Phase 2: Group by size.
	sizeGroups := groupBySize(files)
	if len(sizeGroups) == 0 {
		return nil, nil
	}

	// Phase 3+4+5: For each size group, dedup inodes, partial hash, full hash.
	numWorkers := max(min(runtime.NumCPU()*hashCPUMult, maxHashWorkers), minHashWorkers)
	sem := make(chan struct{}, numWorkers)

	var mu sync.Mutex
	var allGroups []DupeGroup
	var wg sync.WaitGroup

	for size, group := range sizeGroups {
		sem <- struct{}{}
		wg.Add(1)
		go func(sz int64, grp []FileEntry) {
			defer wg.Done()
			defer func() { <-sem }()

			dupes := processSizeGroup(sz, grp)

			if len(dupes) > 0 {
				mu.Lock()
				allGroups = append(allGroups, dupes...)
				mu.Unlock()
			}
		}(size, group)
	}

	wg.Wait()

	// Phase 6: Sort by reclaimable space descending.
	sort.Slice(allGroups, func(i, j int) bool {
		return allGroups[i].ReclaimableBytes() > allGroups[j].ReclaimableBytes()
	})

	return allGroups, nil
}

// processSizeGroup handles phases 3-5 for a single size group.
func processSizeGroup(size int64, files []FileEntry) []DupeGroup {
	// Phase 3: Deduplicate by inode.
	files = deduplicateByInode(files)
	if len(files) < 2 {
		return nil
	}

	// Phase 4: Partial hash — group by partial hash.
	partialGroups := make(map[string][]FileEntry)
	for _, f := range files {
		h, err := hashFilePartial(f.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bw dupes: partial hash %s: %v\n", f.Path, err)
			continue
		}
		partialGroups[h] = append(partialGroups[h], f)
	}

	// Phase 5: Full hash — confirm duplicates within each partial group.
	var result []DupeGroup

	for _, pGroup := range partialGroups {
		if len(pGroup) < 2 {
			continue
		}

		fullGroups := make(map[string][]FileEntry)
		for _, f := range pGroup {
			h, err := hashFileFull(f.Path, f.Size)
			if err != nil {
				fmt.Fprintf(os.Stderr, "bw dupes: full hash %s: %v\n", f.Path, err)
				continue
			}
			fullGroups[h] = append(fullGroups[h], f)
		}

		for hash, fGroup := range fullGroups {
			if len(fGroup) < 2 {
				continue
			}
			// Sort files within group by path for deterministic output.
			sort.Slice(fGroup, func(i, j int) bool {
				return fGroup[i].Path < fGroup[j].Path
			})
			result = append(result, DupeGroup{
				Hash:  hash,
				Size:  size,
				Files: fGroup,
			})
		}
	}

	return result
}
