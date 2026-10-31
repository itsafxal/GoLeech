package media

import (
	"os"
	"testing"
)

func TestApplyNameSwap(t *testing.T) {
	orig := "Ubuntu.22.04.LTS.iso"
	res := ApplyNameSwap(orig, "22.04:24.04")
	if res != "Ubuntu.24.04.LTS.iso" {
		t.Errorf("expected 'Ubuntu.24.04.LTS.iso', got '%s'", res)
	}

	regexSwap := ApplyNameSwap("Movie.2020.1080p.mkv", `20\d\d:2026`)
	if regexSwap != "Movie.2026.1080p.mkv" {
		t.Errorf("expected 'Movie.2026.1080p.mkv', got '%s'", regexSwap)
	}

	noMatch := ApplyNameSwap("Sample.mp4", "invalid")
	if noMatch != "Sample.mp4" {
		t.Errorf("expected 'Sample.mp4', got '%s'", noMatch)
	}
}

func TestSplitFileCustom(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "split_test_*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	data := make([]byte, 100)
	_, _ = tmpFile.Write(data)
	tmpFile.Close()

	// Normal split (max 40): parts should be 40, 40, 20
	parts, err := SplitFileCustom(tmpFile.Name(), 40, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, p := range parts {
			if p != tmpFile.Name() {
				_ = os.Remove(p)
			}
		}
	}()

	if len(parts) != 3 {
		t.Errorf("expected 3 parts for normal split, got %d", len(parts))
	}

	// Equal split (max 40 of 100 bytes => ceil(100/40) = 3 parts => 34 bytes each): parts should be 34, 34, 32
	partsEqual, err := SplitFileCustom(tmpFile.Name(), 40, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, p := range partsEqual {
			if p != tmpFile.Name() {
				_ = os.Remove(p)
			}
		}
	}()

	if len(partsEqual) != 3 {
		t.Errorf("expected 3 parts for equal split, got %d", len(partsEqual))
	}
}
