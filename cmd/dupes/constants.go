//go:build darwin

package main

import "time"

const (
	// Hash settings.
	partialHashSize = 4096       // 4KB for partial hash (fast rejection).
	hashBufferSize  = 128 * 1024 // 128KB streaming read buffer.

	// Worker pool limits.
	minScanWorkers = 8
	maxScanWorkers = 64
	scanCPUMult    = 4

	// Hasher concurrency.
	minHashWorkers = 4
	maxHashWorkers = 32
	hashCPUMult    = 2

	// Manifest filename inside conserve directory.
	manifestFilename = ".burrow-manifest.json"

	// Timeouts.
	trashTimeout = 30 * time.Second
)

// skipDirs are directories never descended into during duplicate scanning.
var skipDirs = map[string]bool{
	// VCS.
	".git": true,
	".svn": true,
	".hg":  true,

	// Package managers / build output.
	"node_modules":     true,
	"__pycache__":      true,
	".pytest_cache":    true,
	"venv":             true,
	".venv":            true,
	"vendor":           true,
	"target":           true,
	".gradle":          true,
	".m2":              true,
	".cargo":           true,
	"Pods":             true,
	"DerivedData":      true,
	"site-packages":    true,
	".tox":             true,
	".npm":             true,
	".pnpm-store":      true,
	".next":            true,
	".nuxt":            true,
	"bower_components": true,

	// System / macOS.
	".Spotlight-V100":         true,
	".fseventsd":              true,
	".DocumentRevisions-V100": true,
	".TemporaryItems":         true,
	".Trash":                  true,
	"__MACOSX":                true,

	// Containers / VMs.
	".docker":     true,
	".containerd": true,
	".lima":       true,
	".colima":     true,
	".orbstack":   true,
}

// skipSystemDirs are top-level directories skipped when scanning /.
var skipSystemRootDirs = map[string]bool{
	"dev":     true,
	"tmp":     true,
	"private": true,
	"cores":   true,
	"net":     true,
	"home":    true,
	"System":  true,
	"sbin":    true,
	"bin":     true,
	"etc":     true,
	"var":     true,
	"Volumes": true,
	"Network": true,
	".vol":    true,
}

// ANSI color codes.
const (
	colorReset      = "\033[0m"
	colorBold       = "\033[1m"
	colorDim        = "\033[2m"
	colorRed        = "\033[38;5;167m"
	colorGreen      = "\033[38;5;108m"
	colorYellow     = "\033[38;5;214m"
	colorBlue       = "\033[38;5;110m"
	colorPurple     = "\033[38;5;139m"
	colorCyan       = "\033[38;5;73m"
	colorGray       = "\033[38;5;241m"
	colorPurpleBold = "\033[1;38;5;139m"
)
