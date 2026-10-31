package utils

import (
	"testing"
)

func TestParseContentDisposition(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Standard quoted filename",
			header:   `attachment; filename="The.Matrix.1999.1080p.mkv"`,
			expected: "The.Matrix.1999.1080p.mkv",
		},
		{
			name:     "Standard unquoted filename",
			header:   `attachment; filename=document.pdf`,
			expected: "document.pdf",
		},
		{
			name:     "RFC 5987 UTF-8 encoded with spaces",
			header:   `attachment; filename*=UTF-8''%5BCK%5D%20Lanja%20Bhoomi%20%282023%29.mkv`,
			expected: "[CK] Lanja Bhoomi (2023).mkv",
		},
		{
			name:     "RFC 5987 with language code",
			header:   `attachment; filename*=utf-8'en'%5BCC%5D%20Avatar%202.mkv`,
			expected: "[CC] Avatar 2.mkv",
		},
		{
			name:     "Both standard and RFC 5987 (RFC 5987 takes precedence)",
			header:   `attachment; filename="fallback.mkv"; filename*=UTF-8''RealName.mkv`,
			expected: "RealName.mkv",
		},
		{
			name:     "Path traversal defense",
			header:   `attachment; filename="../../../etc/passwd"`,
			expected: "passwd",
		},
		{
			name:     "Empty header",
			header:   "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseContentDisposition(tc.header)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestSniffFileExtension(t *testing.T) {
	// Matroska magic bytes
	mkvMagic := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x93, 0x42, 0x82, 0x88, 'm', 'a', 't', 'r', 'o', 's', 'k', 'a'}
	if ext := SniffFileExtension(mkvMagic, ""); ext != ".mkv" {
		t.Errorf("expected .mkv, got %s", ext)
	}

	// MP4 magic bytes: [0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm']
	mp4Magic := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}
	if ext := SniffFileExtension(mp4Magic, ""); ext != ".mp4" {
		t.Errorf("expected .mp4, got %s", ext)
	}

	// ZIP magic bytes: PK\x03\x04
	zipMagic := []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00}
	if ext := SniffFileExtension(zipMagic, ""); ext != ".zip" {
		t.Errorf("expected .zip, got %s", ext)
	}

	// MIME type fallback
	if ext := SniffFileExtension(nil, "video/x-matroska"); ext != ".mkv" {
		t.Errorf("expected .mkv from mime, got %s", ext)
	}
	if ext := SniffFileExtension(nil, "application/pdf; charset=utf-8"); ext != ".pdf" {
		t.Errorf("expected .pdf from mime, got %s", ext)
	}
}
