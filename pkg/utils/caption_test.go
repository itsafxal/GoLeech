package utils

import (
	"strings"
	"testing"
)

func TestApplyPrefixSuffix(t *testing.T) {
	// Case 1: User's exact scenario where suffix is ".mkv" and file already has ".mkv"
	orig := "[CK] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv"
	res := ApplyPrefixSuffix(orig, "[CC]", ".mkv")
	expected := "[CC] [CK] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv"
	if res != expected {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected, res)
	}

	// Case 2: Suffix placed before extension
	res2 := ApplyPrefixSuffix("sample.mp4", "[MyGroup]", " [1080p]")
	expected2 := "[MyGroup] sample [1080p].mp4"
	if res2 != expected2 {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected2, res2)
	}

	// Case 3: Suffix ending with extension e.g. " [1080p].mp4" on .mp4
	res3 := ApplyPrefixSuffix("sample.mp4", "", " [1080p].mp4")
	expected3 := "sample [1080p].mp4"
	if res3 != expected3 {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected3, res3)
	}

	// Case 4: Escape \s space handling
	res4 := ApplyPrefixSuffix("movie.mkv", "@mychannel\\s-", "\\s[HQ]")
	expected4 := "@mychannel - movie [HQ].mkv"
	if res4 != expected4 {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected4, res4)
	}

	// Case 5: Idempotency (running again does not duplicate prefix or suffix)
	res5 := ApplyPrefixSuffix(expected, "[CC]", ".mkv")
	if res5 != expected {
		t.Errorf("ApplyPrefixSuffix idempotency failed:\nExpected: %s\nGot:      %s", expected, res5)
	}
}

func TestDetectQuality(t *testing.T) {
	q1 := DetectQuality("[CK] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv", 0, 0)
	if !strings.EqualFold(q1, "HDRip") && !strings.EqualFold(q1, "HQ") {
		t.Errorf("expected HDRip or HQ, got %s", q1)
	}

	q2 := DetectQuality("Inception.2010.1080p.BluRay.x264.mkv", 1920, 1080)
	if !strings.EqualFold(q2, "1080p") && !strings.EqualFold(q2, "BluRay") {
		t.Errorf("expected 1080p or BluRay, got %s", q2)
	}

	q3 := DetectQuality("video.mp4", 3840, 2160)
	if q3 != "4K" {
		t.Errorf("expected 4K, got %s", q3)
	}
}

func TestFormatCaption(t *testing.T) {
	filename := "[CC] [CK] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv"
	size := int64(270152499) // ~257.64 MB
	tmpl := "{filename} size {size} quality {quality}"

	out := FormatCaption(tmpl, filename, size, 1, 1, "CyberTGz", "123456", 0, 0, 0)
	if !strings.Contains(out, filename) {
		t.Errorf("expected output to contain filename, got: %s", out)
	}
	if !strings.Contains(out, "257.64 MB") {
		t.Errorf("expected output to contain '257.64 MB', got: %s", out)
	}
	if !strings.Contains(out, "HDRip") && !strings.Contains(out, "HQ") {
		t.Errorf("expected output to contain HDRip, got: %s", out)
	}
	if strings.Contains(out, "{filename}") || strings.Contains(out, "{size}") || strings.Contains(out, "{quality}") {
		t.Errorf("unparsed template tokens remaining: %s", out)
	}

	// Test default caption when template is empty
	defaultOut := FormatCaption("", filename, size, 1, 1, "CyberTGz", "123456", 0, 0, 0)
	if !strings.Contains(defaultOut, "<code>"+filename+"</code>") {
		t.Errorf("default caption missing code tag: %s", defaultOut)
	}
	if !strings.Contains(defaultOut, "257.64 MB") {
		t.Errorf("default caption missing size: %s", defaultOut)
	}
	if !strings.Contains(defaultOut, "@CyberTGz") {
		t.Errorf("default caption missing username: %s", defaultOut)
	}
}
