package http

import (
	"net/http"
	"testing"
)

func TestDetermineFilename(t *testing.T) {
	// Case 1: Custom filename takes precedence
	resp := &http.Response{Header: make(http.Header)}
	if name := DetermineFilename(resp, "https://example.com/file.mkv", "Custom_Movie.mkv"); name != "Custom_Movie.mkv" {
		t.Errorf("expected Custom_Movie.mkv, got %s", name)
	}

	// Case 2: RFC 5987 / 6266 Content-Disposition header
	resp2 := &http.Response{Header: make(http.Header)}
	resp2.Header.Set("Content-Disposition", `attachment; filename*=UTF-8''%5BCK%5D%20Lanja%20Bhoomi%20%282023%29.mkv`)
	if name := DetermineFilename(resp2, "https://example.com/attachment.php?id=123", ""); name != "[CK] Lanja Bhoomi (2023).mkv" {
		t.Errorf("expected [CK] Lanja Bhoomi (2023).mkv, got %s", name)
	}

	// Case 3: Standard quoted Content-Disposition
	resp3 := &http.Response{Header: make(http.Header)}
	resp3.Header.Set("Content-Disposition", `attachment; filename="Spider-Man.2023.1080p.mkv"`)
	if name := DetermineFilename(resp3, "https://example.com/download?token=xyz", ""); name != "Spider-Man.2023.1080p.mkv" {
		t.Errorf("expected Spider-Man.2023.1080p.mkv, got %s", name)
	}

	// Case 4: Dynamic script with query parameter (?file=sample.zip)
	resp4 := &http.Response{Header: make(http.Header)}
	if name := DetermineFilename(resp4, "https://example.com/download.php?file=sample.zip&auth=123", ""); name != "sample.zip" {
		t.Errorf("expected sample.zip, got %s", name)
	}

	// Case 5: Direct URL path
	resp5 := &http.Response{Header: make(http.Header)}
	if name := DetermineFilename(resp5, "https://cdn.example.com/media/videos/holiday_2024.mp4", ""); name != "holiday_2024.mp4" {
		t.Errorf("expected holiday_2024.mp4, got %s", name)
	}
}
