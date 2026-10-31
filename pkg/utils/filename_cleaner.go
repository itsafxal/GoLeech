package utils

import (
	"fmt"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Regex for Telegram usernames: @username
	usernameRegex = regexp.MustCompile(`(?i)(?:^|[\s_\[\(\{])(@[a-zA-Z0-9_]{3,})\b`)

	// Regex for Telegram t.me links
	tmeRegex = regexp.MustCompile(`(?i)(?:https?://)?t\.me/[a-zA-Z0-9_+]+`)

	// Regex for website domains in filenames (e.g. www.TamilBlasters.vip, TamilMV.re)
	domainRegex = regexp.MustCompile(`(?i)\b(?:www\.)?[a-zA-Z0-9_-]+\.(?:com|org|net|in|me|co|cc|io|to|biz|info|site|vip|link|xyz|club|online|top|tv|re|pm|am|yt|ws|un|is)\b`)

	// Meaningful patterns to preserve inside brackets/tags
	yearRegex       = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
	seasonEpRegex   = regexp.MustCompile(`(?i)\b(s\d{1,3}(?:e\d{1,3})?|season\s*\d+|ep(?:isode)?\s*\d+|part\s*\d+)\b`)
	resQualityRegex = regexp.MustCompile(`(?i)\b(2160p|1440p|1080p|720p|540p|480p|360p|4k|2k|uhd|fhd|hd|sd|bluray|blu-ray|bdrip|brrip|web-dl|webdl|webrip|web|hdrip|hd-rip|dvdrip|camrip|hdcam|hdtc|remux|proper|repack|x264|x265|h264|h265|hevc|av1|10bit|8bit|hdr10\+?|hdr|dv|sdr)\b`)
	audioCodecRegex = regexp.MustCompile(`(?i)\b(aac|ac3|eac3|ddp?\+?|dts(?:-hd)?|truehd|atmos|flac|mp3|opus|5\.1|7\.1|2\.0)\b`)
	languageRegex   = regexp.MustCompile(`(?i)\b(tamil|telugu|hindi|english|malayalam|kannada|bengali|marathi|gujarati|punjabi|urdu|korean|japanese|chinese|spanish|french|german|russian|italian|dual\s*audio|multi\s*audio|multi\s*sub|esub|subs?)\b`)
	sizePatternRegex = regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*(?:mb|gb)\b`)
)

// IsMeaningfulTag returns true if the content of a tag represents legitimate media information
func IsMeaningfulTag(tag string) bool {
	t := strings.TrimSpace(tag)
	if t == "" {
		return false
	}
	if yearRegex.MatchString(t) {
		return true
	}
	if seasonEpRegex.MatchString(t) {
		return true
	}
	if resQualityRegex.MatchString(t) {
		return true
	}
	if audioCodecRegex.MatchString(t) {
		return true
	}
	if languageRegex.MatchString(t) {
		return true
	}
	if sizePatternRegex.MatchString(t) {
		return true
	}
	return false
}

// ExtractCoreToken strips brackets, braces, parentheses, quotes, and symbols from a prefix
func ExtractCoreToken(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\s", " ")
	return strings.Trim(p, "[]{}()<>'\"@ \t-_:")
}

// DeduplicatePrefix removes existing instances of the configured prefix from the
// filename (regardless of whether it is wrapped in [], {}, (), quotes, or trailing separators).
func DeduplicatePrefix(filename string, userPrefix string) string {
	coreToken := ExtractCoreToken(userPrefix)
	if coreToken == "" {
		return filename
	}

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	escaped := regexp.QuoteMeta(coreToken)
	re := regexp.MustCompile(fmt.Sprintf(`(?i)^\s*(?:[\[\{\(\<\'\"]\s*%s\s*[\]\}\)\>\'\"]|@?%s\b)\s*[-_:]?\s*`, escaped, escaped))

	for {
		loc := re.FindStringIndex(base)
		if loc == nil || loc[0] != 0 {
			break
		}
		base = strings.TrimSpace(base[loc[1]:])
	}

	return base + ext
}

// CleanMediaFilename removes unwanted attribution, usernames, website watermarks,
// and redundant duplicate source tags while strictly preserving meaningful content
// (movie/series titles, years, resolutions, seasons, qualities, languages, and hyphens).
func CleanMediaFilename(filename string, userPrefix string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "file"
	}

	// 1. Sanitize filesystem path if needed
	filename = filepath.Base(filename)

	// 2. Resolve compound / duplicate extensions (e.g. .mkv.mkv -> .mkv)
	ext := filepath.Ext(filename)
	lowerName := strings.ToLower(filename)
	for _, knownExt := range []string{".mkv", ".mp4", ".avi", ".mov", ".zip", ".rar", ".7z", ".tar.gz", ".tar"} {
		dup := knownExt + knownExt
		if strings.HasSuffix(lowerName, dup) {
			filename = filename[:len(filename)-len(knownExt)]
			ext = knownExt
			break
		}
	}
	base := strings.TrimSuffix(filename, ext)

	// 3. Deduplicate configured user prefix (e.g. [CC] or {CC} when prefix is [CC])
	if userPrefix != "" {
		coreToken := ExtractCoreToken(userPrefix)
		if coreToken != "" {
			escaped := regexp.QuoteMeta(coreToken)
			rePref := regexp.MustCompile(fmt.Sprintf(`(?i)^\s*(?:[\[\{\(\<\'\"]\s*%s\s*[\]\}\)\>\'\"]|@?%s\b)\s*[-_:]?\s*`, escaped, escaped))
			for {
				loc := rePref.FindStringIndex(base)
				if loc == nil || loc[0] != 0 {
					break
				}
				base = strings.TrimSpace(base[loc[1]:])
			}
		}
	}

	// 4. Remove leading Telegram usernames (@username)
	reLeadingUser := regexp.MustCompile(`(?i)^\s*(?:[\[\{\(]\s*)?@[a-zA-Z0-9_]{3,}(?:\s*[\]\}\)])?\s*[-_:]?\s*`)
	for {
		loc := reLeadingUser.FindStringIndex(base)
		if loc == nil || loc[0] != 0 {
			break
		}
		base = strings.TrimSpace(base[loc[1]:])
	}

	// 5. Remove website watermark tags: [www.site.com], {site.com}, etc.
	reBrackWeb := regexp.MustCompile(`(?i)[\[\{\(]\s*(?:www\.)?[a-zA-Z0-9_-]+\.(?:com|org|net|in|me|co|cc|io|to|biz|info|site|vip|link|xyz|club|online|top|tv|re|pm|am|yt|ws|un|is)\s*[\]\}\)]\s*[-_:]?\s*`)
	base = reBrackWeb.ReplaceAllString(base, "")

	reUnbrackWeb := regexp.MustCompile(`(?i)^\s*(?:www\.)?[a-zA-Z0-9_-]+\.(?:com|org|net|in|me|co|cc|io|to|biz|info|site|vip|link|xyz|club|online|top|tv|re|pm|am|yt|ws|un|is)\s*[-_:]?\s*`)
	base = reUnbrackWeb.ReplaceAllString(base, "")

	// 6. Remove leading source tags in [ ], { }, ( ), or quotes if not meaningful
	reBrackets := regexp.MustCompile(`^\s*([\[\{\(])([^\[\]\{\}\(\)]{1,35})([\]\}\)])\s*[-_:]?\s*`)
	for {
		m := reBrackets.FindStringSubmatch(base)
		if len(m) > 2 {
			inner := strings.TrimSpace(m[2])
			if !IsMeaningfulTag(inner) {
				loc := reBrackets.FindStringIndex(base)
				if loc != nil && loc[0] == 0 {
					base = strings.TrimSpace(base[loc[1]:])
					continue
				}
			}
		}
		break
	}

	// 7. Remove leading single/double quoted attribution tags (e.g. 'Source' or "Source") if not meaningful
	reQuotes := regexp.MustCompile(`^\s*['"]([^'"]{1,35})['"]\s*[-_:]?\s*`)
	for {
		m := reQuotes.FindStringSubmatch(base)
		if len(m) > 1 {
			inner := strings.TrimSpace(m[1])
			if !IsMeaningfulTag(inner) {
				loc := reQuotes.FindStringIndex(base)
				if loc != nil && loc[0] == 0 {
					base = strings.TrimSpace(base[loc[1]:])
					continue
				}
			}
		}
		break
	}

	// 8. Remove trailing or isolated @username attribution
	reEndUser := regexp.MustCompile(`(?i)\s*[-_:]?\s*(?:[\[\{\(]\s*)?@[a-zA-Z0-9_]{3,}(?:\s*[\]\}\)])?\s*$`)
	base = reEndUser.ReplaceAllString(base, "")

	// 9. Trim dangling leading/trailing separators while preserving internal hyphens
	base = strings.Trim(base, " \t\r\n-_.")
	base = regexp.MustCompile(`\s{2,}`).ReplaceAllString(base, " ")

	if strings.TrimSpace(base) == "" {
		base = strings.TrimSuffix(filename, ext)
	}
	if strings.TrimSpace(base) == "" {
		base = "file"
	}

	return base + ext
}

// DetectLanguages parses language identifiers from filename
func DetectLanguages(filename string) string {
	matches := languageRegex.FindAllString(filename, -1)
	if len(matches) == 0 {
		return ""
	}

	var langs []string
	seen := make(map[string]bool)
	for _, m := range matches {
		clean := strings.Title(strings.ToLower(strings.TrimSpace(m)))
		if !seen[clean] {
			seen[clean] = true
			langs = append(langs, clean)
		}
	}
	return strings.Join(langs, ", ")
}

// DetectSubtitles detects subtitle tags from filename
func DetectSubtitles(filename string) string {
	lower := strings.ToLower(filename)
	if strings.Contains(lower, "esub") || strings.Contains(lower, "esubs") {
		return "English (Esubs)"
	}
	if strings.Contains(lower, "msub") || strings.Contains(lower, "multi sub") {
		return "Multi Subtitles"
	}
	if strings.Contains(lower, "sub") || strings.Contains(lower, "subtitle") {
		return "Available"
	}
	return ""
}

// DetectMimeType determines the MIME type of a file based on its extension
func DetectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mkv":
		return "video/x-matroska"
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".flv":
		return "video/x-flv"
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/x-rar-compressed"
	case ".7z":
		return "application/x-7z-compressed"
	case ".tar":
		return "application/x-tar"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		if t := mime.TypeByExtension(ext); t != "" {
			return strings.Split(t, ";")[0]
		}
		return "application/octet-stream"
	}
}
