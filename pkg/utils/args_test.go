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

	// Extended flags test
	extendedInput := `magnet:?xt=urn:btih:xyz -m "Movies/2026" -ca mp3 -cv mp4 -ns "old:new" -meta "GoLeech Release" -ff "-preset fast" -tl 00:10:00 -d -j -ad -rcf "--transfers 8"`
	extOpts := ParseTaskOptions(extendedInput)

	if extOpts.Subfolder != "Movies/2026" {
		t.Errorf("expected Subfolder 'Movies/2026', got '%s'", extOpts.Subfolder)
	}
	if extOpts.ConvertAudio != "mp3" {
		t.Errorf("expected ConvertAudio 'mp3', got '%s'", extOpts.ConvertAudio)
	}
	if extOpts.ConvertVideo != "mp4" {
		t.Errorf("expected ConvertVideo 'mp4', got '%s'", extOpts.ConvertVideo)
	}
	if extOpts.NameSwap != "old:new" {
		t.Errorf("expected NameSwap 'old:new', got '%s'", extOpts.NameSwap)
	}
	if extOpts.MetadataTitle != "GoLeech Release" {
		t.Errorf("expected MetadataTitle 'GoLeech Release', got '%s'", extOpts.MetadataTitle)
	}
	if extOpts.FFmpegArgs != "-preset fast" {
		t.Errorf("expected FFmpegArgs '-preset fast', got '%s'", extOpts.FFmpegArgs)
	}
	if extOpts.TrimLength != "00:10:00" {
		t.Errorf("expected TrimLength '00:10:00', got '%s'", extOpts.TrimLength)
	}
	if !extOpts.DownloadOnly {
		t.Errorf("expected DownloadOnly true")
	}
	if !extOpts.JoinParts {
		t.Errorf("expected JoinParts true")
	}
	if !extOpts.AudioOnly {
		t.Errorf("expected AudioOnly true")
	}
	if extOpts.RcloneFlags != "--transfers 8" {
		t.Errorf("expected RcloneFlags '--transfers 8', got '%s'", extOpts.RcloneFlags)
	}
}
