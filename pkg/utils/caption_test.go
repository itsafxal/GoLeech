package utils

import (
	"strings"
	"testing"
)

func TestCleanMediaFilename(t *testing.T) {
	// 1. User's exact prompt example:
	// Original: [CC] The.Last.Thing.He.Told.Me.S02E01.1080p.10bit.WEBRip.6CH.x265.mkv
	// Expected: [CC] The Last Thing He Told Me S02E01 1080p 10bit WEBRip 6CH x265.mkv
	f1 := CleanMediaFilename("[CC] The.Last.Thing.He.Told.Me.S02E01.1080p.10bit.WEBRip.6CH.x265.mkv", "")
	expected1 := "[CC] The Last Thing He Told Me S02E01 1080p 10bit WEBRip 6CH x265.mkv"
	if f1 != expected1 {
		t.Errorf("Prompt example 1 failed:\nExpected: %q\nGot:      %q", expected1, f1)
	}

	// 2. Underscores replaced with proper spaces:
	f2 := CleanMediaFilename("The_Last_Thing_He_Told_Me_S02E01_1080p_10bit_WEBRip_6CH_x265.mkv", "")
	expected2 := "The Last Thing He Told Me S02E01 1080p 10bit WEBRip 6CH x265.mkv"
	if f2 != expected2 {
		t.Errorf("Underscore replacement failed:\nExpected: %q\nGot:      %q", expected2, f2)
	}

	// 3. Technical decimals (5.1, H.264) preserved:
	f3 := CleanMediaFilename("Movie.2023.DDP5.1.Atmos.H.264.mkv", "")
	expected3 := "Movie 2023 DDP5.1 Atmos H.264.mkv"
	if f3 != expected3 {
		t.Errorf("Technical decimal preservation failed:\nExpected: %q\nGot:      %q", expected3, f3)
	}

	// 4. Filename containing @username (removed):
	f4 := CleanMediaFilename("@TamilMovies Leo (2023) [1080p].mkv", "")
	if strings.Contains(f4, "@TamilMovies") || !strings.Contains(f4, "Leo (2023) [1080p].mkv") {
		t.Errorf("failed to clean @username: got %q", f4)
	}

	// 5. Filename containing {CC} with configured prefix [CC] (deduplicated):
	f5 := CleanMediaFilename("{CC} Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv", "[CC]")
	expected5 := "Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv"
	if f5 != expected5 {
		t.Errorf("failed to deduplicate {CC}: expected %q, got %q", expected5, f5)
	}

	// 6. Filename containing [CC] [CC] multiple duplicate prefixes:
	f6 := CleanMediaFilename("[CC] [CC] Movie Name 2025.mkv", "[CC]")
	expected6 := "Movie Name 2025.mkv"
	if f6 != expected6 {
		t.Errorf("failed to deduplicate multiple [CC]: expected %q, got %q", expected6, f6)
	}

	// 7. Filename containing unnecessary quoted text:
	f7 := CleanMediaFilename("\"CinemaLover\" 'HDReleases' Spider-Man (2021).mp4", "")
	expected7 := "Spider-Man (2021).mp4"
	if f7 != expected7 {
		t.Errorf("failed to clean quoted text: expected %q, got %q", expected7, f7)
	}

	// 8. Filename with meaningful hyphens (preserved):
	f8 := CleanMediaFilename("Ali-Baba-aur-40-Bhoot-2025.mkv", "")
	expected8 := "Ali-Baba-aur-40-Bhoot-2025.mkv"
	if f8 != expected8 {
		t.Errorf("failed to preserve hyphens: expected %q, got %q", expected8, f8)
	}

	// 9. Filename with season and episode info and source tag:
	f9 := CleanMediaFilename("{SourceTag} Game of Thrones S01E09 1080p BluRay.mkv", "")
	if !strings.Contains(f9, "S01E09") || strings.Contains(f9, "{SourceTag}") {
		t.Errorf("failed season/episode check: got %q", f9)
	}

	// 10. Filename with duplicate compound extension:
	f10 := CleanMediaFilename("Movie.Name.2024.1080p.mkv.mkv", "")
	expected10 := "Movie Name 2024 1080p.mkv"
	if f10 != expected10 {
		t.Errorf("failed to fix duplicate extension: expected %q, got %q", expected10, f10)
	}

	// 11. Filename with website watermarks:
	f11 := CleanMediaFilename("[www.TamilBlasters.vip] - Leo (2023) [1080p].mkv", "")
	expected11 := "Leo (2023) [1080p].mkv"
	if f11 != expected11 {
		t.Errorf("failed to clean website watermark: expected %q, got %q", expected11, f11)
	}
}

func TestApplyPrefixSuffix(t *testing.T) {
	// Case 1: Original prompt example with [CC] prefix configured on dot filename:
	res1 := ApplyPrefixSuffix("The.Last.Thing.He.Told.Me.S02E01.1080p.10bit.WEBRip.6CH.x265.mkv", "[CC]", "")
	expected1 := "[CC] The Last Thing He Told Me S02E01 1080p 10bit WEBRip 6CH x265.mkv"
	if res1 != expected1 {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected1, res1)
	}

	// Case 2: [CC] already in filename - do NOT duplicate:
	res2 := ApplyPrefixSuffix("[CC] The.Last.Thing.He.Told.Me.S02E01.1080p.10bit.WEBRip.6CH.x265.mkv", "[CC]", "")
	if res2 != expected1 {
		t.Errorf("ApplyPrefixSuffix deduplication failed:\nExpected: %s\nGot:      %s", expected1, res2)
	}

	// Case 3: Preserving [CK] while adding [CC] prefix:
	res3 := ApplyPrefixSuffix("[CK] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv", "[CC]", ".mkv")
	expected3 := "[CC] [CK] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv"
	if res3 != expected3 {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected3, res3)
	}

	// Case 4: Suffix placed before extension:
	res4 := ApplyPrefixSuffix("sample.mp4", "[MyGroup]", " [1080p]")
	expected4 := "[MyGroup] sample [1080p].mp4"
	if res4 != expected4 {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected4, res4)
	}

	// Case 5: Suffix ending with extension e.g. " [1080p].mp4" on .mp4:
	res5 := ApplyPrefixSuffix("sample.mp4", "", " [1080p].mp4")
	expected5 := "sample [1080p].mp4"
	if res5 != expected5 {
		t.Errorf("ApplyPrefixSuffix failed:\nExpected: %s\nGot:      %s", expected5, res5)
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

func TestCustomCaptionTemplate(t *testing.T) {
	filename := "[CC] The Last Thing He Told Me S02E01 1080p 10bit WEBRip 6CH x265.mkv"
	size := int64(1548576000)
	tmpl := `🎬 <b>{filename}</b>
📦 <b>Size:</b> {size}
🎞 <b>Quality:</b> {quality}
🌐 <b>Languages:</b> {languages}
💬 <b>Subtitles:</b> {subtitles}`

	ctx := CaptionContext{
		Filename:    filename,
		Size:        size,
		PartNum:     1,
		TotalParts:  1,
		Username:    "@CyberTGz",
		UserID:      "123456",
		Width:       1920,
		Height:      1080,
		DurationSec: 3600,
		Languages:   "English",
		Subtitles:   "English",
		FileID:      "task_01",
	}

	out := FormatCaption(tmpl, ctx)
	if !strings.Contains(out, filename) {
		t.Errorf("expected caption to contain filename, got: %s", out)
	}
	if !strings.Contains(out, "1.44 GB") {
		t.Errorf("expected caption to contain '1.44 GB', got: %s", out)
	}
	if !strings.Contains(out, "1080p") {
		t.Errorf("expected caption to contain '1080p', got: %s", out)
	}
	if strings.Contains(out, "{filename}") || strings.Contains(out, "{size}") {
		t.Errorf("unparsed template tokens remaining: %s", out)
	}
}
