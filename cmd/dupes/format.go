//go:build darwin

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// humanizeBytes converts bytes to human-readable string using SI units.
func humanizeBytes(size int64) string {
	if size < 0 {
		return "0 B"
	}
	const unit = 1000
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	value := float64(size) / float64(div)
	return fmt.Sprintf("%.1f %cB", value, "kMGTPE"[exp])
}

// parseSize parses a human-readable size string (e.g., "1KB", "500MB", "1024").
func parseSize(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)

	multipliers := map[string]int64{
		"KB": 1000,
		"MB": 1000000,
		"GB": 1000000000,
		"TB": 1000000000000,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(upper, suffix) {
			numStr := s[:len(s)-len(suffix)]
			if numStr == "" {
				return 0, fmt.Errorf("no number before unit: %s", s)
			}
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size number: %s", s)
			}
			if n < 0 {
				return 0, fmt.Errorf("negative size: %s", s)
			}
			return n * mult, nil
		}
	}

	// Plain number (bytes).
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("negative size: %s", s)
	}
	return n, nil
}

// formatReport produces human-readable duplicate report output.
func formatReport(groups []DupeGroup) string {
	if len(groups) == 0 {
		return colorGreen + "No duplicates found." + colorReset + "\n"
	}

	var b strings.Builder
	var totalReclaimable int64
	var totalDupeFiles int

	for i, g := range groups {
		reclaimable := g.ReclaimableBytes()
		totalReclaimable += reclaimable
		totalDupeFiles += len(g.Files) - 1

		fmt.Fprintf(&b, "\n%s%d copies%s (%s each, %s%s reclaimable%s):\n",
			colorBold, len(g.Files), colorReset,
			humanizeBytes(g.Size),
			colorYellow, humanizeBytes(reclaimable), colorReset)

		for j, f := range g.Files {
			var marker string
			if j == 0 {
				marker = colorGreen + "  ✓" + colorReset
			} else {
				marker = colorGray + "   " + colorReset
			}
			fmt.Fprintf(&b, "%s [%d] %s\n", marker, j+1, displayPath(f.Path))
		}

		if i < len(groups)-1 {
			b.WriteString("\n")
		}
	}

	fmt.Fprintf(&b, "\n%s━━━ Summary ━━━%s\n", colorPurpleBold, colorReset)
	fmt.Fprintf(&b, "  %d duplicate groups, %d redundant files\n", len(groups), totalDupeFiles)
	fmt.Fprintf(&b, "  %sReclaimable: %s%s\n", colorYellow, humanizeBytes(totalReclaimable), colorReset)

	return b.String()
}

// jsonReport is the JSON output structure.
type jsonReport struct {
	Groups           []jsonGroup `json:"groups"`
	TotalGroups      int         `json:"total_groups"`
	TotalDupeFiles   int         `json:"total_duplicate_files"`
	ReclaimableBytes int64       `json:"reclaimable_bytes"`
	ReclaimableHuman string      `json:"reclaimable_human"`
}

type jsonGroup struct {
	Hash             string     `json:"hash"`
	Size             int64      `json:"size"`
	SizeHuman        string     `json:"size_human"`
	Copies           int        `json:"copies"`
	ReclaimableBytes int64      `json:"reclaimable_bytes"`
	Files            []jsonFile `json:"files"`
}

type jsonFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// formatJSON produces machine-readable JSON output.
func formatJSON(groups []DupeGroup) string {
	report := jsonReport{
		Groups: make([]jsonGroup, 0, len(groups)),
	}

	var totalReclaimable int64
	var totalDupeFiles int

	for _, g := range groups {
		reclaimable := g.ReclaimableBytes()
		totalReclaimable += reclaimable
		totalDupeFiles += len(g.Files) - 1

		jg := jsonGroup{
			Hash:             g.Hash,
			Size:             g.Size,
			SizeHuman:        humanizeBytes(g.Size),
			Copies:           len(g.Files),
			ReclaimableBytes: reclaimable,
			Files:            make([]jsonFile, 0, len(g.Files)),
		}

		for _, f := range g.Files {
			jg.Files = append(jg.Files, jsonFile{Path: f.Path, Size: f.Size})
		}

		report.Groups = append(report.Groups, jg)
	}

	report.TotalGroups = len(groups)
	report.TotalDupeFiles = totalDupeFiles
	report.ReclaimableBytes = totalReclaimable
	report.ReclaimableHuman = humanizeBytes(totalReclaimable)

	data, _ := json.MarshalIndent(report, "", "  ")
	return string(data)
}
