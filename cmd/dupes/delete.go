//go:build darwin

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// runDeleteMode presents each group interactively, asking which file to keep.
func runDeleteMode(groups []DupeGroup) error {
	if len(groups) == 0 {
		fmt.Println("No duplicates to delete.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	var totalReclaimed int64
	var totalDeleted int

	for i, g := range groups {
		reclaimable := g.ReclaimableBytes()

		fmt.Printf("\n%s[%d/%d]%s %s%d copies%s (%s each, %s%s reclaimable%s):\n",
			colorGray, i+1, len(groups), colorReset,
			colorBold, len(g.Files), colorReset,
			humanizeBytes(g.Size),
			colorYellow, humanizeBytes(reclaimable), colorReset)

		for j, f := range g.Files {
			fmt.Printf("    [%d] %s\n", j+1, displayPath(f.Path))
		}

		fmt.Printf("\n  Keep which? [1-%d, s=skip, q=quit, a=auto-keep-first]: ", len(g.Files))

		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(input)

		switch strings.ToLower(input) {
		case "s":
			continue
		case "q":
			fmt.Printf("\nStopped. Deleted %d files, reclaimed %s.\n",
				totalDeleted, humanizeBytes(totalReclaimed))
			return nil
		case "a":
			// Auto-keep first for this and all remaining groups.
			for _, rg := range groups[i:] {
				toDelete := buildDeleteList(rg, 0)
				deleted, reclaimed := trashFiles(toDelete)
				totalDeleted += deleted
				totalReclaimed += reclaimed
			}
			fmt.Printf("\nAuto-deleted remaining. Deleted %d files, reclaimed %s.\n",
				totalDeleted, humanizeBytes(totalReclaimed))
			return nil
		default:
			keepIdx, err := parseKeeperChoice(input, len(g.Files))
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Invalid choice, skipping group.\n")
				continue
			}

			toDelete := buildDeleteList(g, keepIdx)
			deleted, reclaimed := trashFiles(toDelete)
			totalDeleted += deleted
			totalReclaimed += reclaimed
		}
	}

	fmt.Printf("\n%sDone.%s Deleted %d files, reclaimed %s.\n",
		colorGreen, colorReset, totalDeleted, humanizeBytes(totalReclaimed))
	return nil
}

// parseKeeperChoice parses user input "1"-"N" to a 0-based index.
func parseKeeperChoice(input string, numFiles int) (int, error) {
	n, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("not a number: %s", input)
	}
	if n < 1 || n > numFiles {
		return 0, fmt.Errorf("out of range: %d (must be 1-%d)", n, numFiles)
	}
	return n - 1, nil
}

// buildDeleteList returns paths of all files except the keeper.
func buildDeleteList(g DupeGroup, keepIdx int) []string {
	var paths []string
	for i, f := range g.Files {
		if i != keepIdx {
			paths = append(paths, f.Path)
		}
	}
	return paths
}

// trashFiles moves files to Finder Trash, returning count and bytes reclaimed.
func trashFiles(paths []string) (int, int64) {
	var deleted int
	var reclaimed int64

	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", displayPath(path), err)
			continue
		}
		size := info.Size()

		if err := moveToTrash(path); err != nil {
			fmt.Fprintf(os.Stderr, "  failed to trash %s: %v\n", displayPath(path), err)
			continue
		}

		deleted++
		reclaimed += size
		fmt.Printf("  %s✓%s trashed %s\n", colorGreen, colorReset, displayPath(path))
	}

	return deleted, reclaimed
}

// moveToTrash uses macOS Finder to move a file/directory to Trash.
func moveToTrash(path string) error {
	if err := validatePath(path); err != nil {
		return err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	if err := validatePath(absPath); err != nil {
		return err
	}

	escapedPath := strings.ReplaceAll(absPath, "\\", "\\\\")
	escapedPath = strings.ReplaceAll(escapedPath, "\"", "\\\"")

	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, escapedPath)

	ctx, cancel := context.WithTimeout(context.Background(), trashTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("timeout moving to Trash")
		}
		return fmt.Errorf("failed to move to Trash: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// validatePath checks path safety for external commands.
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null bytes")
	}
	if slices.Contains(strings.Split(path, string(filepath.Separator)), "..") {
		return fmt.Errorf("path contains traversal components: %s", path)
	}
	return nil
}
