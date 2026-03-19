//go:build darwin

package main

import (
	"testing"
)

func TestParseSizeFlag(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1KB", 1000, false},
		{"1MB", 1000000, false},
		{"1GB", 1000000000, false},
		{"0", 0, false},
		{"", 0, true},
		{"xyz", 0, true},
	}

	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("parseSize(%q) expected error", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("parseSize(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestMutuallyExclusive(t *testing.T) {
	tests := []struct {
		deleteMode  bool
		conserveDir string
		restoreDir  string
		wantErr     bool
	}{
		{false, "", "", false},          // report mode
		{true, "", "", false},           // delete only
		{false, "/tmp/c", "", false},    // conserve only
		{false, "", "/tmp/r", false},    // restore only
		{true, "/tmp/c", "", true},      // delete + conserve
		{true, "", "/tmp/r", true},      // delete + restore
		{false, "/tmp/c", "/tmp/r", true}, // conserve + restore
		{true, "/tmp/c", "/tmp/r", true},  // all three
	}

	for _, tt := range tests {
		err := validateModeFlags(tt.deleteMode, tt.conserveDir, tt.restoreDir)
		if tt.wantErr && err == nil {
			t.Errorf("validateModeFlags(%v, %q, %q) expected error",
				tt.deleteMode, tt.conserveDir, tt.restoreDir)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("validateModeFlags(%v, %q, %q) unexpected error: %v",
				tt.deleteMode, tt.conserveDir, tt.restoreDir, err)
		}
	}
}
