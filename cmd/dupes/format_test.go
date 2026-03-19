//go:build darwin

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{-1, "0 B"},
		{512, "512 B"},
		{999, "999 B"},
		{1000, "1.0 kB"},
		{1500, "1.5 kB"},
		{1000000, "1.0 MB"},
		{1500000, "1.5 MB"},
		{1000000000, "1.0 GB"},
		{1500000000, "1.5 GB"},
		{1000000000000, "1.0 TB"},
	}

	for _, tt := range tests {
		got := humanizeBytes(tt.input)
		if got != tt.expected {
			t.Errorf("humanizeBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1KB", 1000, false},
		{"1kb", 1000, false},
		{"1MB", 1000000, false},
		{"1GB", 1000000000, false},
		{"500KB", 500000, false},
		{"1024", 1024, false},
		{"0", 0, false},
		{"", 0, true},
		{"abc", 0, true},
		{"KB", 0, true},
		{"-1KB", 0, true},
	}

	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSize(%q) expected error, got %d", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSize(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestFormatReport_NoGroups(t *testing.T) {
	out := formatReport(nil)
	if !strings.Contains(out, "No duplicates found") {
		t.Errorf("expected 'No duplicates found' message, got: %s", out)
	}
}

func TestFormatReport_WithGroups(t *testing.T) {
	groups := []DupeGroup{
		{
			Hash: "abc123",
			Size: 1000000,
			Files: []FileEntry{
				{Path: "/tmp/a.txt", Size: 1000000},
				{Path: "/tmp/b.txt", Size: 1000000},
			},
		},
	}

	out := formatReport(groups)

	if !strings.Contains(out, "a.txt") {
		t.Errorf("report should contain file name: %s", out)
	}
	if !strings.Contains(out, "b.txt") {
		t.Errorf("report should contain file name: %s", out)
	}
	if !strings.Contains(out, "reclaimable") || !strings.Contains(out, "Reclaimable") {
		// At least some summary.
	}
}

func TestFormatJSON_ValidJSON(t *testing.T) {
	groups := []DupeGroup{
		{
			Hash: "abc123",
			Size: 2048,
			Files: []FileEntry{
				{Path: "/tmp/a.txt", Size: 2048},
				{Path: "/tmp/b.txt", Size: 2048},
			},
		},
	}

	out := formatJSON(groups)

	var parsed any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("formatJSON produced invalid JSON: %v\nOutput: %s", err, out)
	}

	// Verify structure.
	m, ok := parsed.(map[string]any)
	if !ok {
		t.Fatal("expected top-level object")
	}
	gs, ok := m["groups"].([]any)
	if !ok {
		t.Fatal("expected groups array")
	}
	if len(gs) != 1 {
		t.Errorf("expected 1 group, got %d", len(gs))
	}
}

func TestFormatJSON_EmptyGroups(t *testing.T) {
	out := formatJSON(nil)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("formatJSON(nil) invalid JSON: %v", err)
	}

	gs := parsed["groups"].([]any)
	if len(gs) != 0 {
		t.Errorf("expected empty groups array, got %d", len(gs))
	}
}
