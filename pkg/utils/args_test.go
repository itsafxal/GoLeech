package utils

import (
	"testing"
)

func TestParseTaskOptions(t *testing.T) {
	input := `https://example.com/movie.mp4 -n "My Movie.mp4" -doc -z -sv -f -sp 1.5G -up "gdrive:backup"`
	opts := ParseTaskOptions(input)

	if opts.Link != "https://example.com/movie.mp4" {
		t.Errorf("expected link 'https://example.com/movie.mp4', got '%s'", opts.Link)
	}
	if opts.Name != "My Movie.mp4" {
		t.Errorf("expected name 'My Movie.mp4', got '%s'", opts.Name)
	}
	if !opts.AsDocument {
		t.Errorf("expected AsDocument true")
	}
	if !opts.Zip {
		t.Errorf("expected Zip true")
	}
	if !opts.SampleVideo {
		t.Errorf("expected SampleVideo true")
	}
	if !opts.ForceStart {
		t.Errorf("expected ForceStart true")
	}
	if opts.UploadDest != "gdrive:backup" {
		t.Errorf("expected UploadDest 'gdrive:backup', got '%s'", opts.UploadDest)
	}
	expectedSplit := int64(1.5 * 1024 * 1024 * 1024)
	if opts.SplitSize != expectedSplit {
		t.Errorf("expected SplitSize %d, got %d", expectedSplit, opts.SplitSize)
	}
}
