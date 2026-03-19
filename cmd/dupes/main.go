//go:build darwin

// Package main provides the bw dupes command for finding and managing duplicate files.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func run() error {
	deleteMode := flag.Bool("delete", false, "interactive deletion mode (Finder Trash)")
	conserveDir := flag.String("conserve", "", "move duplicates to conserve directory")
	restoreDir := flag.String("restore", "", "restore conserved files from manifest")
	filterFile := flag.String("file", "", "restore a single file (use with --restore)")
	minSizeStr := flag.String("min-size", "1KB", "skip files under this size (e.g., 1KB, 1MB, 1GB)")
	jsonOutput := flag.Bool("json", false, "machine-readable JSON output")
	help := flag.Bool("help", false, "show help")
	flag.Usage = usage
	flag.Parse()

	if *help {
		usage()
		return nil
	}

	if err := validateModeFlags(*deleteMode, *conserveDir, *restoreDir); err != nil {
		return err
	}

	// Restore mode doesn't need a scan directory.
	if *restoreDir != "" {
		return runRestoreMode(*restoreDir, *filterFile)
	}

	// Determine scan directory.
	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	minSize, err := parseSize(*minSizeStr)
	if err != nil {
		return fmt.Errorf("invalid --min-size: %w", err)
	}

	cfg := ScanConfig{
		Root:        absDir,
		MinSize:     minSize,
		ConserveDir: *conserveDir,
	}

	fmt.Fprintf(os.Stderr, "bw dupes: scanning %s (min-size: %s)...\n",
		displayPath(absDir), humanizeBytes(minSize))

	groups, err := findDuplicates(cfg)
	if err != nil {
		return err
	}

	// Dispatch to mode.
	switch {
	case *deleteMode:
		return runDeleteMode(groups)
	case *conserveDir != "":
		return runConserveMode(groups, *conserveDir, absDir)
	case *jsonOutput:
		fmt.Println(formatJSON(groups))
		return nil
	default:
		fmt.Print(formatReport(groups))
		return nil
	}
}

// validateModeFlags checks that --delete, --conserve, and --restore are mutually exclusive.
func validateModeFlags(deleteMode bool, conserveDir, restoreDir string) error {
	count := 0
	if deleteMode {
		count++
	}
	if conserveDir != "" {
		count++
	}
	if restoreDir != "" {
		count++
	}
	if count > 1 {
		return fmt.Errorf("--delete, --conserve, and --restore are mutually exclusive")
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `bw dupes — find and manage duplicate files

Usage:
  bw dupes [directory]              Report duplicates (default: current dir)
  bw dupes [directory] --json       Machine-readable output
  bw dupes [directory] --delete     Interactive deletion (Finder Trash)
  bw dupes [directory] --conserve <dir>  Move duplicates preserving paths
  bw dupes --restore <dir>          Restore conserved files
  bw dupes --restore <dir> --file <path>  Restore a single file

Flags:
`)
	flag.PrintDefaults()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bw dupes: %v\n", err)
		os.Exit(1)
	}
}
