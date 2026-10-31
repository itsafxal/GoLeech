package downloader

import (
	"testing"
)

func TestClassifySourceDirectLink(t *testing.T) {
	tests := []struct {
		name     string
		rawLink  string
		expected SourceType
	}{
		{
			name:     "Direct HTTP file",
			rawLink:  "http://example.com/movie.mkv",
			expected: SourceTypeDirectURL,
		},
		{
			name:     "Direct HTTPS file with query",
			rawLink:  "https://cdn.example.org/files/video.mp4?token=abc&expires=123",
			expected: SourceTypeDirectURL,
		},
		{
			name:     "Dynamic PHP attachment",
			rawLink:  "https://example.com/attachment.php?id=99281",
			expected: SourceTypeDirectURL,
		},
		{
			name:     "Magnet link",
			rawLink:  "magnet:?xt=urn:btih:d6b63d76b328a9b2b5275a593e8e19b023f03a67&dn=Ubuntu",
			expected: SourceTypeTorrent,
		},
		{
			name:     "Torrent URL",
			rawLink:  "https://releases.ubuntu.com/22.04/ubuntu-22.04.torrent",
			expected: SourceTypeTorrent,
		},
		{
			name:     "YouTube video link",
			rawLink:  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: SourceTypeYTDLP,
		},
		{
			name:     "Google Drive file link",
			rawLink:  "https://drive.google.com/file/d/1234567890/view",
			expected: SourceTypeGDrive,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := ClassifySource(nil, nil, tc.rawLink)
			if src.Type != tc.expected {
				t.Errorf("expected %s, got %s for %s", tc.expected, src.Type, tc.rawLink)
			}
		})
	}
}

func TestDetectEngine(t *testing.T) {
	if eng := DetectEngine("https://example.com/file.mkv"); eng != "Aria2" {
		t.Errorf("expected Aria2, got %s", eng)
	}
	if eng := DetectEngine("magnet:?xt=urn:btih:12345"); eng != "Aria2" {
		t.Errorf("expected Aria2, got %s", eng)
	}
	if eng := DetectEngine("tg://media"); eng != "Telegram" {
		t.Errorf("expected Telegram, got %s", eng)
	}
	if eng := DetectEngine("https://youtu.be/12345"); eng != "yt-dlp" {
		t.Errorf("expected yt-dlp, got %s", eng)
	}
}

