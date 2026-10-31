package utils

import (
	"strings"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{-10, "0 B"},
		{500, "500.00 B"},
		{1024, "1.00 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
	}

	for _, tc := range tests {
		actual := FormatBytes(tc.input)
		if actual != tc.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", tc.input, actual, tc.expected)
		}
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0 B/s"},
		{-5.0, "0 B/s"},
		{1024, "1.00 KB/s"},
		{5242880, "5.00 MB/s"},
	}

	for _, tc := range tests {
		actual := FormatSpeed(tc.input)
		if actual != tc.expected {
			t.Errorf("FormatSpeed(%f) = %s, expected %s", tc.input, actual, tc.expected)
		}
	}
}

func TestFormatSeconds(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0s"},
		{45, "45s"},
		{65, "1m 5s"},
		{3665, "1h 1m 5s"},
		{90065, "1d 1h 1m 5s"},
	}

	for _, tc := range tests {
		actual := FormatSeconds(tc.input)
		if actual != tc.expected {
			t.Errorf("FormatSeconds(%d) = %s, expected %s", tc.input, actual, tc.expected)
		}
	}
}

func TestProgressBar(t *testing.T) {
	b0 := ProgressBar(0)
	if !strings.Contains(b0, "□") || strings.Contains(b0, "■") {
		t.Errorf("ProgressBar(0) unexpected: %s", b0)
	}

	b100 := ProgressBar(100)
	if !strings.Contains(b100, "■") || strings.Contains(b100, "□") {
		t.Errorf("ProgressBar(100) unexpected: %s", b100)
	}

	b50 := ProgressBar(50)
	if !strings.Contains(b50, "■") || !strings.Contains(b50, "□") {
		t.Errorf("ProgressBar(50) unexpected: %s", b50)
	}
}

func TestCleanFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.mp4", "normal.mp4"},
		{"invalid:file?.mkv", "invalid_file_.mkv"},
		{"../path/traversal.zip", "traversal.zip"},
		{"", "file"},
	}

	for _, tc := range tests {
		actual := CleanFilename(tc.input)
		if actual != tc.expected {
			t.Errorf("CleanFilename(%s) = %s, expected %s", tc.input, actual, tc.expected)
		}
	}
}
