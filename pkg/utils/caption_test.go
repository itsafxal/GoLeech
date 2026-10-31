package utils

import (
	"strings"
	"testing"
)

func TestCleanMediaFilename(t *testing.T) {
	// 1. Filename containing @username
	f1 := CleanMediaFilename("@TamilMovies Leo (2023) [1080p].mkv", "")
	if strings.Contains(f1, "@TamilMovies") || !strings.Contains(f1, "Leo (2023) [1080p].mkv") {
		t.Errorf("failed to clean @username: got %q", f1)
	}

	// 2. Filename containing {CC} with configured prefix [CC]
	f2 := CleanMediaFilename("{CC} Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv", "[CC]")
	expected2 := "Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv"
	if f2 != expected2 {
		t.Errorf("failed to deduplicate {CC}: expected %q, got %q", expected2, f2)
	}

	// 3. Filename containing [CC] [CC] multiple duplicate prefixes
	f3 := CleanMediaFilename("[CC] [CC] Movie Name 2025.mkv", "[CC]")
	expected3 := "Movie Name 2025.mkv"
	if f3 != expected3 {
		t.Errorf("failed to deduplicate multiple [CC]: expected %q, got %q", expected3, f3)
	}

	// 4. Filename containing unnecessary quoted text
	f4 := CleanMediaFilename("\"CinemaLover\" 'HDReleases' Spider-Man (2021).mp4", "")
	expected4 := "Spider-Man (2021).mp4"
	if f4 != expected4 {
		t.Errorf("failed to clean quoted text: expected %q, got %q", expected4, f4)
	}

	// 5. Filename with meaningful hyphens (must be preserved!)
	f5 := CleanMediaFilename("Ali-Baba-aur-40-Bhoot-2025.mkv", "[CC]")
	expected5 := "Ali-Baba-aur-40-Bhoot-2025.mkv"
	if f5 != expected5 {
		t.Errorf("failed to preserve hyphens: expected %q, got %q", expected5, f5)
	}

	// 6. Filename with season and episode information
	f6 := CleanMediaFilename("{SourceTag} Game of Thrones S01E09 1080p BluRay.mkv", "")
	if !strings.Contains(f6, "S01E09") || strings.Contains(f6, "{SourceTag}") {
		t.Errorf("failed season/episode check: got %q", f6)
	}

	// 7. Filename with Unicode characters
	f7 := CleanMediaFilename("[TG] లంచ భూమి (2023) Tamil HQ HDRip.mkv", "")
	if !strings.Contains(f7, "లంచ భూమి (2023)") || strings.Contains(f7, "[TG]") {
		t.Errorf("failed Unicode handling: got %q", f7)
	}

	// 8. Filename with duplicate compound extension
	f8 := CleanMediaFilename("Movie.Name.2024.1080p.mkv.mkv", "")
	expected8 := "Movie.Name.2024.1080p.mkv"
	if f8 != expected8 {
		t.Errorf("failed to fix duplicate extension: expected %q, got %q", expected8, f8)
	}

	// 9. Filename with website watermarks
	f9 := CleanMediaFilename("[www.TamilBlasters.vip] - Leo (2023) [1080p].mkv", "")
	expected9 := "Leo (2023) [1080p].mkv"
	if f9 != expected9 {
		t.Errorf("failed to clean website watermark: expected %q, got %q", expected9, f9)
	}
}

func TestApplyPrefixSuffix(t *testing.T) {
	// Case 1: User's exact scenario from prompt:
	// Incoming: {CC} Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv
	// Configured prefix: [CC]
	// Expected: [CC] Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv
	orig := "{CC} Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv"
	res := ApplyPrefixSuffix(orig, "[CC]", "")
	expected := "[CC] Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv"
	if res != expected {
		t.Errorf("ApplyPrefixSuffix duplicate {CC} failed:\nExpected: %s\nGot:      %s", expected, res)
	}

	// Case 2: Another exact scenario from prompt:
	// Incoming: [CC] [CC] Movie Name 2025.mkv
	// Configured prefix: [CC]
	// Expected: [CC] Movie Name 2025.mkv
	orig2 := "[CC] [CC] Movie Name 2025.mkv"
	res2 := ApplyPrefixSuffix(orig2, "[CC]", "")
	expected2 := "[CC] Movie Name 2025.mkv"
	if res2 != expected2 {
		t.Errorf("ApplyPrefixSuffix multiple [CC] failed:\nExpected: %s\nGot:      %s", expected2, res2)
	}

	// Case 3: Suffix ending with extension e.g. ".mkv" on ".mkv"
	res3 := ApplyPrefixSuffix("[CK] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv", "[CC]", ".mkv")
	expected3 := "[CC] Lanja Bhoomi (2023) Tamil HQ HDRip - x264 - AAC - 250MB.mkv"
	if res3 != expected3 {
		t.Errorf("ApplyPrefixSuffix duplicate .mkv failed:\nExpected: %s\nGot:      %s", expected3, res3)
	}

	// Case 4: Meaningful hyphens with prefix
	res4 := ApplyPrefixSuffix("Ali-Baba-aur-40-Bhoot-2025.mkv", "[CC]", "")
	expected4 := "[CC] Ali-Baba-aur-40-Bhoot-2025.mkv"
	if res4 != expected4 {
		t.Errorf("ApplyPrefixSuffix hyphens failed:\nExpected: %s\nGot:      %s", expected4, res4)
	}

	// Case 5: Suffix placed before extension
	res5 := ApplyPrefixSuffix("sample.mp4", "[MyGroup]", " [1080p]")
	expected5 := "[MyGroup] sample [1080p].mp4"
	if res5 != expected5 {
		t.Errorf("ApplyPrefixSuffix suffix position failed:\nExpected: %s\nGot:      %s", expected5, res5)
	}
}

func TestCustomCaptionTemplate(t *testing.T) {
	ctx := CaptionContext{
		Filename:    "[CC] Ali Baba aur 40 Bhoot S01E29 Kaali Jheel.mkv",
		Size:        int64(270152499), // ~257.64 MB
		PartNum:     1,
		TotalParts:  1,
		Username:    "CyberTGz",
		UserID:      "987654321",
		Width:       1920,
		Height:      1080,
		DurationSec: 3725, // 1h 2m 5s
		Quality:     "1080p",
		Languages:   "Hindi",
		Subtitles:   "English",
		MimeType:    "video/x-matroska",
		FileID:      "57A5F3E0",
	}

	tmpl := `📁 File Name: {filename}

📦 Size: {size}
⏱ Duration: {duration}
🎬 Quality: {quality}
🌐 Language: {language}
💬 Subtitles: {subtitles}
🆔 File ID: {file_id}
📄 MIME: {mime_type}`

	out := FormatCaption(tmpl, ctx)

	// Check each placeholder is resolved properly
	if !strings.Contains(out, ctx.Filename) {
		t.Errorf("missing filename in caption: %s", out)
	}
	if !strings.Contains(out, "257.64 MB") {
		t.Errorf("missing size in caption: %s", out)
	}
	if !strings.Contains(out, "1h 2m 5s") {
		t.Errorf("missing duration in caption: %s", out)
	}
	if !strings.Contains(out, "1080p") {
		t.Errorf("missing quality in caption: %s", out)
	}
	if !strings.Contains(out, "Hindi") {
		t.Errorf("missing language in caption: %s", out)
	}
	if !strings.Contains(out, "English") {
		t.Errorf("missing subtitles in caption: %s", out)
	}
	if !strings.Contains(out, "57A5F3E0") {
		t.Errorf("missing file_id in caption: %s", out)
	}
	if !strings.Contains(out, "video/x-matroska") {
		t.Errorf("missing mime_type in caption: %s", out)
	}

	// Verify no raw unparsed placeholders remain
	for _, ph := range []string{"{filename}", "{size}", "{duration}", "{quality}", "{language}", "{subtitles}", "{file_id}", "{mime_type}"} {
		if strings.Contains(out, ph) {
			t.Errorf("unparsed placeholder %s found in output: %s", ph, out)
		}
	}

	// Test missing metadata fallback
	emptyCtx := CaptionContext{
		Filename: "sample.mp4",
		Size:     1024,
	}
	outEmpty := FormatCaption(tmpl, emptyCtx)
	if strings.Contains(outEmpty, "{duration}") || strings.Contains(outEmpty, "{language}") || strings.Contains(outEmpty, "{subtitles}") {
		t.Errorf("unresolved tokens in fallback caption: %s", outEmpty)
	}
	if !strings.Contains(outEmpty, "N/A") {
		t.Errorf("expected N/A for missing duration/language: %s", outEmpty)
	}
}
