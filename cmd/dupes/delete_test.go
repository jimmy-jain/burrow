//go:build darwin

package main

import (
	"path/filepath"
	"testing"
)

func TestValidatePath_Valid(t *testing.T) {
	tests := []string{
		"/Users/jimmy/file.txt",
		"/tmp/test",
		"/var/log/syslog",
	}

	for _, path := range tests {
		if err := validatePath(path); err != nil {
			t.Errorf("validatePath(%q) unexpected error: %v", path, err)
		}
	}
}

func TestValidatePath_Invalid(t *testing.T) {
	tests := []struct {
		path string
		desc string
	}{
		{"", "empty path"},
		{"relative/path", "relative path"},
		{"/path/with\x00null", "null bytes"},
		{"/path/../traversal", "path traversal"},
	}

	for _, tt := range tests {
		if err := validatePath(tt.path); err == nil {
			t.Errorf("validatePath(%q) expected error for %s", tt.path, tt.desc)
		}
	}
}

func TestRunDeleteMode_NoGroups(t *testing.T) {
	// Delete mode with no groups should be a no-op.
	err := runDeleteMode(nil)
	if err != nil {
		t.Errorf("runDeleteMode(nil) unexpected error: %v", err)
	}
}

func TestSelectKeeper_ValidInput(t *testing.T) {
	tests := []struct {
		input    string
		numFiles int
		expected int
	}{
		{"1", 3, 0},
		{"2", 3, 1},
		{"3", 3, 2},
	}

	for _, tt := range tests {
		got, err := parseKeeperChoice(tt.input, tt.numFiles)
		if err != nil {
			t.Errorf("parseKeeperChoice(%q, %d) unexpected error: %v", tt.input, tt.numFiles, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parseKeeperChoice(%q, %d) = %d, want %d", tt.input, tt.numFiles, got, tt.expected)
		}
	}
}

func TestSelectKeeper_Invalid(t *testing.T) {
	tests := []struct {
		input    string
		numFiles int
	}{
		{"0", 3},
		{"4", 3},
		{"-1", 3},
		{"abc", 3},
	}

	for _, tt := range tests {
		_, err := parseKeeperChoice(tt.input, tt.numFiles)
		if err == nil {
			t.Errorf("parseKeeperChoice(%q, %d) expected error", tt.input, tt.numFiles)
		}
	}
}

func TestBuildDeleteList(t *testing.T) {
	group := DupeGroup{
		Hash: "abc",
		Size: 1024,
		Files: []FileEntry{
			{Path: filepath.Join("/tmp", "a.txt"), Size: 1024},
			{Path: filepath.Join("/tmp", "b.txt"), Size: 1024},
			{Path: filepath.Join("/tmp", "c.txt"), Size: 1024},
		},
	}

	// Keep index 0 (first file), delete indices 1 and 2.
	toDelete := buildDeleteList(group, 0)

	if len(toDelete) != 2 {
		t.Fatalf("expected 2 files to delete, got %d", len(toDelete))
	}
	if toDelete[0] != group.Files[1].Path {
		t.Errorf("expected %s, got %s", group.Files[1].Path, toDelete[0])
	}
	if toDelete[1] != group.Files[2].Path {
		t.Errorf("expected %s, got %s", group.Files[2].Path, toDelete[1])
	}
}
