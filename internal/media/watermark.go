package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// WatermarkConfig holds configuration parameters for periodic video text overlay
type WatermarkConfig struct {
	Enabled      bool   `json:"enabled"`
	Text         string `json:"text"`
	Position     string `json:"position"` // bottom_center, bottom_left, bottom_right, top_center, top_left, top_right
	FontSize     int    `json:"font_size"`
	FontColor    string `json:"font_color"`
	BgColor      string `json:"bg_color"`
	DurationSec  int    `json:"duration_sec"`  // how long watermark is visible
	IntervalSec  int    `json:"interval_sec"`  // recurrence interval
	InitialDelay int    `json:"initial_delay"` // delay before first appearance
}

// NewWatermarkConfigFromMap parses user settings ExtraData map into WatermarkConfig
func NewWatermarkConfigFromMap(get func(string) string) WatermarkConfig {
	cfg := WatermarkConfig{
		Enabled:      get("WATERMARK_ENABLED") == "true",
		Text:         get("WATERMARK_TEXT"),
		Position:     get("WATERMARK_POS"),
		FontSize:     24,
		FontColor:    "white",
		BgColor:      "black@0.4",
		DurationSec:  5,
		IntervalSec:  120,
		InitialDelay: 30,
	}

	if cfg.Text == "" {
		cfg.Text = "Join Our Channel"
	}
	if cfg.Position == "" {
		cfg.Position = "bottom_center"
	}
	if s := get("WATERMARK_SIZE"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			cfg.FontSize = v
		}
	}
	if c := get("WATERMARK_COLOR"); c != "" {
		cfg.FontColor = c
	}
	if d := get("WATERMARK_DURATION"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			cfg.DurationSec = v
		}
	}
	if i := get("WATERMARK_INTERVAL"); i != "" {
		if v, err := strconv.Atoi(i); err == nil && v > 0 {
			cfg.IntervalSec = v
		}
	}
	if del := get("WATERMARK_DELAY"); del != "" {
		if v, err := strconv.Atoi(del); err == nil && v >= 0 {
			cfg.InitialDelay = v
		}
	}

	return cfg
}

// ApplyVideoWatermark burns custom text into video frames periodically at configured intervals and positions using FFmpeg
func ApplyVideoWatermark(ctx context.Context, inputPath string, cfg WatermarkConfig, onProgress func(pct float64)) (string, error) {
	if !cfg.Enabled || strings.TrimSpace(cfg.Text) == "" {
		return inputPath, nil
	}

	// Verify input file exists
	fi, err := os.Stat(inputPath)
	if err != nil {
		return "", fmt.Errorf("video file not found: %w", err)
	}
	if fi.IsDir() {
		return inputPath, nil
	}

	// Verify file is a video container
	ext := strings.ToLower(filepath.Ext(inputPath))
	isVideo := (ext == ".mp4" || ext == ".mkv" || ext == ".avi" || ext == ".mov" || ext == ".webm" || ext == ".flv" || ext == ".ts" || ext == ".wmv")
	if !isVideo {
		return inputPath, nil
	}

	// Ensure ffmpeg is installed
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg binary not found in system PATH (required for video watermark overlay)")
	}

	// Set defaults
	if cfg.FontSize <= 0 {
		cfg.FontSize = 24
	}
	if cfg.FontColor == "" {
		cfg.FontColor = "white"
	}
	if cfg.BgColor == "" {
		cfg.BgColor = "black@0.4"
	}
	if cfg.DurationSec <= 0 {
		cfg.DurationSec = 5
	}
	if cfg.IntervalSec <= 0 {
		cfg.IntervalSec = 120
	}
	if cfg.InitialDelay < 0 {
		cfg.InitialDelay = 30
	}
	if cfg.IntervalSec <= cfg.DurationSec {
		cfg.IntervalSec = cfg.DurationSec + 10
	}

	// Position coordinates
	var xExpr, yExpr string
	switch strings.ToLower(cfg.Position) {
	case "bottom_left":
		xExpr = "40"
		yExpr = "h-th-40"
	case "bottom_right":
		xExpr = "w-text_w-40"
		yExpr = "h-th-40"
	case "top_center":
		xExpr = "(w-text_w)/2"
		yExpr = "40"
	case "top_left":
		xExpr = "40"
		yExpr = "40"
	case "top_right":
		xExpr = "w-text_w-40"
		yExpr = "40"
	default: // "bottom_center"
		xExpr = "(w-text_w)/2"
		yExpr = "h-th-40"
	}

	// Periodic enable expression: between(mod(t - delay, interval), 0, duration) * gte(t, delay)
	enableExpr := fmt.Sprintf("between(mod(t-%d,%d),0,%d)*gte(t,%d)",
		cfg.InitialDelay, cfg.IntervalSec, cfg.DurationSec, cfg.InitialDelay)

	dir := filepath.Dir(inputPath)

	// Create temporary UTF-8 text file to support arbitrary Unicode, quotes, colons, emojis safely
	textFileName := fmt.Sprintf("wm_txt_%d.txt", time.Now().UnixNano())
	textFilePath := filepath.Join(dir, textFileName)
	if err := os.WriteFile(textFilePath, []byte(cfg.Text), 0644); err != nil {
		return "", fmt.Errorf("failed to create temporary textfile for watermark: %w", err)
	}
	defer os.Remove(textFilePath)

	// Format textfile path for FFmpeg filter (forward slashes and escaped colon)
	escapedTextPath := strings.ReplaceAll(textFilePath, "\\", "/")
	escapedTextPath = strings.ReplaceAll(escapedTextPath, ":", "\\:")

	// Font parameter detection
	fontParam := ""
	fontCandidates := []string{
		"C:/Windows/Fonts/arial.ttf",
		"C:/Windows/Fonts/segoeui.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
	}
	for _, f := range fontCandidates {
		if _, statErr := os.Stat(filepath.FromSlash(f)); statErr == nil {
			fEscaped := strings.ReplaceAll(f, ":", "\\:")
			fontParam = fmt.Sprintf(":fontfile='%s'", fEscaped)
			break
		}
	}

	filter := fmt.Sprintf(
		"drawtext=textfile='%s'%s:fontsize=%d:fontcolor=%s:box=1:boxcolor=%s:boxborderw=5:shadowcolor=black@0.6:shadowx=2:shadowy=2:x=%s:y=%s:enable='%s'",
		escapedTextPath, fontParam, cfg.FontSize, cfg.FontColor, cfg.BgColor, xExpr, yExpr, enableExpr,
	)

	tempOut := filepath.Join(dir, fmt.Sprintf("wm_out_%d%s", time.Now().UnixNano(), ext))

	// Construct FFmpeg command
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-i", inputPath,
		"-vf", filter,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "22",
		"-c:a", "copy",
		"-map", "0",
		tempOut,
	}

	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tempOut)
		// If failed due to subtitle stream copying in MP4 container, retry with mov_text or without subtitles
		if strings.Contains(string(output), "subtitle") || strings.Contains(string(output), "Could not find tag for codec") {
			retryArgs := []string{
				"-y",
				"-hide_banner",
				"-loglevel", "error",
				"-i", inputPath,
				"-vf", filter,
				"-c:v", "libx264",
				"-preset", "veryfast",
				"-crf", "22",
				"-c:a", "copy",
				"-c:s", "mov_text",
				"-map", "0",
				tempOut,
			}
			cmdRetry := exec.CommandContext(ctx, ffmpegPath, retryArgs...)
			if retryOut, rErr := cmdRetry.CombinedOutput(); rErr != nil {
				_ = os.Remove(tempOut)
				return "", fmt.Errorf("ffmpeg watermark processing failed: %w (log: %s)", rErr, string(retryOut))
			}
		} else {
			return "", fmt.Errorf("ffmpeg watermark processing failed: %w (log: %s)", err, string(output))
		}
	}

	// Verify generated file exists and is non-empty
	outFi, err := os.Stat(tempOut)
	if err != nil || outFi.Size() == 0 {
		_ = os.Remove(tempOut)
		return "", fmt.Errorf("ffmpeg generated an empty or missing output file")
	}

	// Replace original file with watermarked file
	_ = os.Remove(inputPath)
	if err := os.Rename(tempOut, inputPath); err != nil {
		// If rename fails across volumes, return tempOut
		return tempOut, nil
	}

	return inputPath, nil
}
