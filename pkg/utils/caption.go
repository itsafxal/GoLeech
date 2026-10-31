package utils

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	qualityRegex     = regexp.MustCompile(`(?i)\b(2160p|1440p|1080p|720p|540p|480p|360p|4k|2k|hdrip|hd-rip|web-dl|webrip|bluray|brrip|bdrip|dvdrip|camrip|hdcam|cam|hq|hdr|hevc|x264|x265|10bit)\b`)
	templateVarRegex = regexp.MustCompile(`(?i)(\{([a-zA-Z0-9_]+)\}|%\(([a-zA-Z0-9_]+)\))`)
)

// CaptionContext carries comprehensive media details and metadata for template evaluation
type CaptionContext struct {
	Filename    string
	Size        int64
	PartNum     int
	TotalParts  int
	Username    string
	UserID      string
	Width       int
	Height      int
	DurationSec int64
	Quality     string
	Languages   string
	Subtitles   string
	MimeType    string
	OrigCaption string
	FileID      string
}

// ApplyPrefixSuffix transforms a filename by attaching prefix to the start and suffix before the extension.
// It cleans unnecessary tags and prevents duplicate prefixes or extensions.
func ApplyPrefixSuffix(filename, prefix, suffix string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return filename
	}

	// Unescape \s to real spaces
	prefix = strings.ReplaceAll(prefix, "\\s", " ")
	suffix = strings.ReplaceAll(suffix, "\\s", " ")

	// 1. Clean the filename and deduplicate configured prefix
	cleaned := CleanMediaFilename(filename, prefix)

	ext := filepath.Ext(cleaned)
	base := strings.TrimSuffix(cleaned, ext)

	// 2. Apply Prefix
	if prefix != "" {
		cleanPrefix := prefix
		if !strings.HasSuffix(cleanPrefix, " ") {
			cleanPrefix += " "
		}
		trimmedPrefix := strings.TrimSpace(cleanPrefix)
		if trimmedPrefix != "" && !strings.HasPrefix(base, trimmedPrefix) {
			base = cleanPrefix + base
		}
	}

	// 3. Apply Suffix (placed before extension)
	if suffix != "" {
		cleanSuffix := suffix
		// Prevent duplicate extension (e.g. if suffix is ".mkv" or " [1080p].mkv" and ext is ".mkv")
		if ext != "" {
			trimmedSuffix := strings.TrimSpace(cleanSuffix)
			if strings.EqualFold(trimmedSuffix, ext) {
				cleanSuffix = ""
			} else if strings.HasSuffix(strings.ToLower(cleanSuffix), strings.ToLower(ext)) {
				cleanSuffix = cleanSuffix[:len(cleanSuffix)-len(ext)]
			}
		}

		trimmedSuffix := strings.TrimSpace(cleanSuffix)
		if trimmedSuffix != "" && !strings.HasSuffix(base, trimmedSuffix) {
			if !strings.HasPrefix(cleanSuffix, " ") && !strings.HasPrefix(cleanSuffix, "-") && !strings.HasPrefix(cleanSuffix, "_") && !strings.HasPrefix(cleanSuffix, ".") {
				cleanSuffix = " " + cleanSuffix
			}
			base = base + cleanSuffix
		}
	}

	return base + ext
}

// DetectQuality extracts the video resolution and quality profile from metadata and filename tags
func DetectQuality(filename string, width, height int) string {
	var res string
	if height >= 2160 || width >= 3840 {
		res = "4K"
	} else if height >= 1440 || width >= 2560 {
		res = "2K"
	} else if height >= 1080 || width >= 1920 {
		res = "1080p"
	} else if height >= 720 || width >= 1280 {
		res = "720p"
	} else if height >= 480 || width >= 640 {
		res = "480p"
	} else if height >= 360 {
		res = "360p"
	}

	matches := qualityRegex.FindAllString(filename, -1)
	tag := ""
	for _, m := range matches {
		upper := strings.ToUpper(m)
		if upper != "X264" && upper != "X265" && upper != "HEVC" && upper != "10BIT" {
			tag = m
			break
		}
	}

	if tag != "" {
		return tag
	}
	if res != "" {
		return res
	}
	return "HD"
}

// ReplaceTemplateVars replaces {key} and %(key) tokens case-insensitively with provided variables
func ReplaceTemplateVars(template string, vars map[string]string) string {
	return templateVarRegex.ReplaceAllStringFunc(template, func(match string) string {
		sub := templateVarRegex.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		key := sub[2]
		if key == "" {
			key = sub[3]
		}
		key = strings.ToLower(key)
		if val, ok := vars[key]; ok {
			return val
		}
		return match
	})
}

// FormatCaption formats a complete leech caption using standard layout or user template with full metadata support
func FormatCaption(template string, ctx CaptionContext) string {
	quality := ctx.Quality
	if quality == "" {
		quality = DetectQuality(ctx.Filename, ctx.Width, ctx.Height)
	}

	resolution := ""
	if ctx.Width > 0 && ctx.Height > 0 {
		resolution = fmt.Sprintf("%dx%d", ctx.Width, ctx.Height)
	} else if quality != "" {
		resolution = quality
	} else {
		resolution = "HD"
	}

	durationStr := ""
	if ctx.DurationSec > 0 {
		durationStr = FormatSeconds(ctx.DurationSec)
	} else {
		durationStr = "N/A"
	}

	languages := ctx.Languages
	if languages == "" {
		languages = DetectLanguages(ctx.Filename)
	}
	if languages == "" {
		languages = "N/A"
	}

	subtitles := ctx.Subtitles
	if subtitles == "" {
		subtitles = DetectSubtitles(ctx.Filename)
	}
	if subtitles == "" {
		subtitles = "None"
	}

	mimeType := ctx.MimeType
	if mimeType == "" {
		mimeType = DetectMimeType(ctx.Filename)
	}

	fileID := ctx.FileID
	if fileID == "" {
		fileID = "N/A"
	}

	ext := strings.TrimPrefix(filepath.Ext(ctx.Filename), ".")
	if ext == "" {
		ext = "file"
	}

	cleanUsername := ctx.Username
	if cleanUsername != "" && !strings.HasPrefix(cleanUsername, "@") {
		cleanUsername = "@" + cleanUsername
	}

	sizeStr := FormatBytes(ctx.Size)

	// If template is empty, return default standard GoLeech caption
	if strings.TrimSpace(template) == "" {
		var sb strings.Builder
		sb.WriteString("<code>" + html.EscapeString(ctx.Filename) + "</code>\n\n")
		sb.WriteString("┠ <b>ꜱɪᴢᴇ:</b> " + sizeStr)
		if ctx.TotalParts > 1 {
			sb.WriteString(fmt.Sprintf("\n┠ <b>ᴘᴀʀᴛ:</b> %d / %d", ctx.PartNum, ctx.TotalParts))
		}
		if cleanUsername != "" {
			sb.WriteString("\n┠ <b>ʙʏ:</b> " + cleanUsername)
		}
		return truncateCaption(sb.String())
	}

	vars := map[string]string{
		"filename":   ctx.Filename,
		"name":       ctx.Filename,
		"size":       sizeStr,
		"quality":    quality,
		"resolution": resolution,
		"duration":   durationStr,
		"language":   languages,
		"languages":  languages,
		"subtitles":  subtitles,
		"subtitle":   subtitles,
		"subs":       subtitles,
		"caption":    ctx.OrigCaption,
		"mime_type":  mimeType,
		"mimetype":   mimeType,
		"file_id":    fileID,
		"task_id":    fileID,
		"gid":        fileID,
		"user":       cleanUsername,
		"username":   cleanUsername,
		"userid":     ctx.UserID,
		"user_id":    ctx.UserID,
		"part":       fmt.Sprintf("%d", ctx.PartNum),
		"total":      fmt.Sprintf("%d", ctx.TotalParts),
		"ext":        ext,
		"format":     ext,
	}

	formatted := ReplaceTemplateVars(template, vars)

	lowerTmpl := strings.ToLower(template)
	if !strings.Contains(lowerTmpl, "{filename}") && !strings.Contains(lowerTmpl, "{name}") &&
		!strings.Contains(lowerTmpl, "%(filename)") && !strings.Contains(lowerTmpl, "%(name)") {
		var sb strings.Builder
		sb.WriteString("<code>" + html.EscapeString(ctx.Filename) + "</code>\n\n")
		sb.WriteString("┠ <b>ꜱɪᴢᴇ:</b> " + sizeStr)
		if ctx.TotalParts > 1 {
			sb.WriteString(fmt.Sprintf("\n┠ <b>ᴘᴀʀᴛ:</b> %d / %d", ctx.PartNum, ctx.TotalParts))
		}
		if cleanUsername != "" {
			sb.WriteString("\n┠ <b>ʙʏ:</b> " + cleanUsername)
		}
		sb.WriteString("\n\n" + formatted)
		return truncateCaption(sb.String())
	}

	if ctx.TotalParts > 1 && !strings.Contains(lowerTmpl, "{part}") && !strings.Contains(lowerTmpl, "%(part)") {
		formatted += fmt.Sprintf("\n\n┠ <b>ᴘᴀʀᴛ:</b> %d / %d", ctx.PartNum, ctx.TotalParts)
	}

	return truncateCaption(formatted)
}

// FormatCaptionSimple provides backward-compatibility for simple parameter callers
func FormatCaptionSimple(template, filename string, size int64, partNum, totalParts int, username, userID string, width, height int, durationSec int64) string {
	return FormatCaption(template, CaptionContext{
		Filename:    filename,
		Size:        size,
		PartNum:     partNum,
		TotalParts:  totalParts,
		Username:    username,
		UserID:      userID,
		Width:       width,
		Height:      height,
		DurationSec: durationSec,
	})
}

// truncateCaption ensures the final caption string stays within Telegram's 1024-character caption limit
func truncateCaption(s string) string {
	runes := []rune(s)
	if len(runes) <= 1024 {
		return s
	}
	return string(runes[:1021]) + "..."
}
