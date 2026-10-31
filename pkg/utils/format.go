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

// BlockProgressBar generates a high-contrast animated block progress bar like ██████░░░░
func BlockProgressBar(pct float64) string {
	if math.IsNaN(pct) || pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	total := 10
	filled := int(math.Round((pct / 100.0) * float64(total)))
	if filled > total {
		filled = total
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", total-filled)
}

// SpinnerFrames contains Unicode rotation animation frames
var SpinnerFrames = []string{"◐", "◓", "◑", "◒"}

// GetSpinnerFrame returns the rotating animation frame for a given tick
func GetSpinnerFrame(tick int) string {
	if tick < 0 {
		tick = -tick
	}
	return SpinnerFrames[tick%len(SpinnerFrames)]
}

// ParseSize converts human-formatted size strings (e.g. 2GB, 2.5gb, 1000mb, 4000000000) to bytes
func ParseSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}

	var multiplier float64 = 1
	var numStr string

	if strings.HasSuffix(s, "PB") {
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "PB")
	} else if strings.HasSuffix(s, "TB") {
		multiplier = 1024 * 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "TB")
	} else if strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(s, "KB")
	} else if strings.HasSuffix(s, "B") {
		multiplier = 1
		numStr = strings.TrimSuffix(s, "B")
	} else {
		numStr = s
	}

	numStr = strings.TrimSpace(numStr)
	var val float64
	if _, err := fmt.Sscanf(numStr, "%f", &val); err == nil && val > 0 {
		return int64(val * multiplier)
	}

	return 0
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
