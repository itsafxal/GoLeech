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

// ApplyPrefixSuffix transforms a filename by attaching prefix to the start and suffix before the extension.
// It prevents duplicate extensions (e.g. .mkv.mkv) and properly parses \s escape characters.
func ApplyPrefixSuffix(filename, prefix, suffix string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return filename
	}

	// Unescape \s to real spaces
	prefix = strings.ReplaceAll(prefix, "\\s", " ")
	suffix = strings.ReplaceAll(suffix, "\\s", " ")

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	// 1. Apply Prefix
	if prefix != "" {
		cleanPrefix := prefix
		trimmedPrefix := strings.TrimSpace(cleanPrefix)
		if trimmedPrefix != "" && !strings.HasPrefix(base, trimmedPrefix) {
			if !strings.HasSuffix(cleanPrefix, " ") {
				cleanPrefix += " "
			}
			base = cleanPrefix + base
		}
	}

	// 2. Apply Suffix (placed before extension)
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

// FormatCaption formats a complete leech caption using standard layout or user template
func FormatCaption(template, filename string, size int64, partNum, totalParts int, username, userID string, width, height int, durationSec int64) string {
	quality := DetectQuality(filename, width, height)

	resolution := ""
	if width > 0 && height > 0 {
		resolution = fmt.Sprintf("%dx%d", width, height)
	} else if quality != "" {
		resolution = quality
	} else {
		resolution = "HD"
	}

	durationStr := ""
	if durationSec > 0 {
		durationStr = FormatSeconds(durationSec)
	} else {
		durationStr = "N/A"
	}

	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		ext = "file"
	}

	cleanUsername := username
	if cleanUsername != "" && !strings.HasPrefix(cleanUsername, "@") {
		cleanUsername = "@" + cleanUsername
	}

	sizeStr := FormatBytes(size)

	// If template is empty, return default standard GoLeech caption
	if strings.TrimSpace(template) == "" {
		var sb strings.Builder
		sb.WriteString("<code>" + html.EscapeString(filename) + "</code>\n\n")
		sb.WriteString("┠ <b>ꜱɪᴢᴇ:</b> " + sizeStr)
		if totalParts > 1 {
			sb.WriteString(fmt.Sprintf("\n┠ <b>ᴘᴀʀᴛ:</b> %d / %d", partNum, totalParts))
		}
		if cleanUsername != "" {
			sb.WriteString("\n┠ <b>ʙʏ:</b> " + cleanUsername)
		}
		return sb.String()
	}

	vars := map[string]string{
		"filename":   filename,
		"name":       filename,
		"size":       sizeStr,
		"quality":    quality,
		"resolution": resolution,
		"duration":   durationStr,
		"user":       cleanUsername,
		"username":   cleanUsername,
		"userid":     userID,
		"user_id":    userID,
		"part":       fmt.Sprintf("%d", partNum),
		"total":      fmt.Sprintf("%d", totalParts),
		"ext":        ext,
		"format":     ext,
		"languages":  "Default",
		"language":   "Default",
		"subtitles":  "None",
		"subs":       "None",
	}

	formatted := ReplaceTemplateVars(template, vars)

	lowerTmpl := strings.ToLower(template)
	if !strings.Contains(lowerTmpl, "{filename}") && !strings.Contains(lowerTmpl, "{name}") &&
		!strings.Contains(lowerTmpl, "%(filename)") && !strings.Contains(lowerTmpl, "%(name)") {
		var sb strings.Builder
		sb.WriteString("<code>" + html.EscapeString(filename) + "</code>\n\n")
		sb.WriteString("┠ <b>ꜱɪᴢᴇ:</b> " + sizeStr)
		if totalParts > 1 {
			sb.WriteString(fmt.Sprintf("\n┠ <b>ᴘᴀʀᴛ:</b> %d / %d", partNum, totalParts))
		}
		if cleanUsername != "" {
			sb.WriteString("\n┠ <b>ʙʏ:</b> " + cleanUsername)
		}
		sb.WriteString("\n\n" + formatted)
		return sb.String()
	}

	if totalParts > 1 && !strings.Contains(lowerTmpl, "{part}") && !strings.Contains(lowerTmpl, "%(part)") {
		formatted += fmt.Sprintf("\n\n┠ <b>ᴘᴀʀᴛ:</b> %d / %d", partNum, totalParts)
	}

	return formatted
}
