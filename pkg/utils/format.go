package utils

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"
)

var sizeUnits = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// FormatBytes formats byte sizes into human readable strings (e.g. 1.50 GB)
func FormatBytes(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	fSize := float64(size)
	unitIdx := 0
	for fSize >= 1024 && unitIdx < len(sizeUnits)-1 {
		fSize /= 1024
		unitIdx++
	}
	return fmt.Sprintf("%.2f %s", fSize, sizeUnits[unitIdx])
}

// FormatSpeed formats transfer speeds in bytes/sec to human readable format (e.g. 2.50 MB/s)
func FormatSpeed(speed float64) string {
	if speed <= 0 {
		return "0 B/s"
	}
	unitIdx := 0
	for speed >= 1024 && unitIdx < len(sizeUnits)-1 {
		speed /= 1024
		unitIdx++
	}
	return fmt.Sprintf("%.2f %s/s", speed, sizeUnits[unitIdx])
}

// FormatSeconds formats seconds into concise human duration (e.g. 1d 4h 20m 10s)
func FormatSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}

	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	var parts []string
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}

	return strings.Join(parts, " ")
}

// FormatDuration formats time.Duration into concise human duration
func FormatDuration(d time.Duration) string {
	return FormatSeconds(int64(d.Seconds()))
}

// ProgressBar generates a graphical unicode progress bar string for 0-100%
func ProgressBar(pct float64) string {
	if math.IsNaN(pct) || pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	totalBlocks := 12
	filledExact := (pct / 100.0) * float64(totalBlocks)
	fullBlocks := int(filledExact)
	fraction := filledExact - float64(fullBlocks)

	partialChars := []string{"", "▤", "▥", "▦", "▧", "▨", "▩", "■"}
	partIdx := int(fraction * 8.0)
	if partIdx > 7 {
		partIdx = 7
	}

	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < fullBlocks && i < totalBlocks; i++ {
		sb.WriteString("■")
	}

	remaining := totalBlocks - fullBlocks
	if remaining > 0 && partIdx > 0 {
		sb.WriteString(partialChars[partIdx])
		remaining--
	}

	for i := 0; i < remaining; i++ {
		sb.WriteString("□")
	}
	sb.WriteString("]")

	return sb.String()
}

// CleanFilename sanitizes an input filename removing hazardous characters and paths
func CleanFilename(name string) string {
	name = filepath.Base(name)
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	clean := replacer.Replace(name)
	clean = strings.TrimSpace(clean)
	if clean == "" || clean == "." {
		return "file"
	}
	return clean
}
