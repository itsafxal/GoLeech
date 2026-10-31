package utils

import (
	"bytes"
	"mime"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// RFC 5987 / 6266 filename*=charset'lang'encoded_value
	rfc5987Regex = regexp.MustCompile(`(?i)filename\*\s*=\s*(?:([a-zA-Z0-9_\-]+)''|([a-zA-Z0-9_\-]+)'[a-zA-Z0-9_\-]*')?([^;\s]+)`)
	// Standard filename="value" or filename=value
	stdFilenameRegex = regexp.MustCompile(`(?i)filename\s*=\s*(?:"([^"]+)"|'([^']+)'|([^;\s]+))`)
)

// ParseContentDisposition extracts the true filename from a Content-Disposition header,
// supporting standard filename="...", RFC 5987/6266 filename*=UTF-8''..., and unquoting.
func ParseContentDisposition(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}

	// 1. Try RFC 5987/6266 filename*= first (highest precedence per spec)
	if match := rfc5987Regex.FindStringSubmatch(header); len(match) > 3 {
		rawVal := match[3]
		// May have quotes around it
		rawVal = strings.Trim(rawVal, `"'`)
		if decoded, err := url.QueryUnescape(rawVal); err == nil && strings.TrimSpace(decoded) != "" {
			return CleanExtractedFilename(decoded)
		}
		if decoded, err := url.PathUnescape(rawVal); err == nil && strings.TrimSpace(decoded) != "" {
			return CleanExtractedFilename(decoded)
		}
		if rawVal != "" {
			return CleanExtractedFilename(rawVal)
		}
	}

	// 2. Standard filename parameter
	if match := stdFilenameRegex.FindStringSubmatch(header); len(match) > 3 {
		for i := 1; i <= 3; i++ {
			if match[i] != "" {
				name := strings.Trim(match[i], `"' `)
				if name != "" {
					return CleanExtractedFilename(name)
				}
			}
		}
	}

	// 3. Fallback to standard mime parser
	if _, params, err := mime.ParseMediaType(header); err == nil {
		if name, ok := params["filename*"]; ok && name != "" {
			parts := strings.SplitN(name, "''", 2)
			if len(parts) == 2 {
				if dec, err := url.QueryUnescape(parts[1]); err == nil {
					return CleanExtractedFilename(dec)
				}
			}
			return CleanExtractedFilename(name)
		}
		if name, ok := params["filename"]; ok && name != "" {
			return CleanExtractedFilename(name)
		}
	}

	return ""
}

// CleanExtractedFilename cleans path components and illegal characters from a filename
func CleanExtractedFilename(name string) string {
	name = filepath.Base(name)
	name = strings.Trim(name, " \t\r\n\"'")
	name = strings.ReplaceAll(name, "\\", "/")
	parts := strings.Split(name, "/")
	name = parts[len(parts)-1]
	name = strings.Trim(name, " \t\r\n")
	return name
}

// SniffFileExtension inspects magic bytes (first 512 bytes) and MIME type to deduce correct file extension
func SniffFileExtension(data []byte, mimeType string) string {
	// 1. Check Magic Bytes
	if len(data) >= 4 {
		// Matroska / WebM: 1A 45 DF A3
		if bytes.HasPrefix(data, []byte{0x1A, 0x45, 0xDF, 0xA3}) {
			if bytes.Contains(data[:min(len(data), 64)], []byte("webm")) {
				return ".webm"
			}
			return ".mkv"
		}
		// MP4 / MOV: 'ftyp' at offset 4..8
		if len(data) >= 12 && string(data[4:8]) == "ftyp" {
			brand := string(data[8:12])
			if brand == "qt  " {
				return ".mov"
			}
			return ".mp4"
		}
		// RIFF AVI / WEBP:
		if len(data) >= 12 && string(data[0:4]) == "RIFF" {
			if string(data[8:12]) == "AVI " {
				return ".avi"
			}
			if string(data[8:12]) == "WEBP" {
				return ".webp"
			}
		}
		// ZIP: PK\x03\x04
		if bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04}) {
			return ".zip"
		}
		// RAR: Rar!\x1a\x07
		if bytes.HasPrefix(data, []byte{0x52, 0x61, 0x72, 0x21}) {
			return ".rar"
		}
		// 7z: 7z\xbc\xaf\x27\x1c
		if len(data) >= 6 && bytes.HasPrefix(data, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
			return ".7z"
		}
		// PDF: %PDF
		if bytes.HasPrefix(data, []byte("%PDF")) {
			return ".pdf"
		}
		// GZIP: \x1f\x8b
		if bytes.HasPrefix(data, []byte{0x1F, 0x8B}) {
			return ".gz"
		}
		// FLAC: fLaC
		if bytes.HasPrefix(data, []byte("fLaC")) {
			return ".flac"
		}
		// OGG: OggS
		if bytes.HasPrefix(data, []byte("OggS")) {
			return ".ogg"
		}
		// PNG: \x89PNG
		if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
			return ".png"
		}
		// JPEG: \xff\xd8\xff
		if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
			return ".jpg"
		}
		// MP3: ID3 or frame sync FF FB / FF FA
		if bytes.HasPrefix(data, []byte("ID3")) || (data[0] == 0xFF && (data[1]&0xFE) == 0xFA) {
			return ".mp3"
		}
	}

	// 2. MIME type fallback
	cleanMime := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	switch cleanMime {
	case "video/x-matroska":
		return ".mkv"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "video/x-msvideo":
		return ".avi"
	case "video/webm":
		return ".webm"
	case "video/x-flv":
		return ".flv"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/flac":
		return ".flac"
	case "audio/ogg", "audio/opus":
		return ".ogg"
	case "audio/aac":
		return ".aac"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "application/zip":
		return ".zip"
	case "application/x-rar-compressed", "application/vnd.rar", "application/x-rar":
		return ".rar"
	case "application/x-7z-compressed":
		return ".7z"
	case "application/x-tar":
		return ".tar"
	case "application/gzip", "application/x-gzip":
		return ".gz"
	case "application/pdf":
		return ".pdf"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	}

	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
