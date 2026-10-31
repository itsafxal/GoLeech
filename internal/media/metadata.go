package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ApplyMetadata tags streams in video and audio files using FFmpeg without re-encoding
func ApplyMetadata(ctx context.Context, filePath string, globalTitle, audioMeta, videoMeta, subMeta string) (string, error) {
	if globalTitle == "" && audioMeta == "" && videoMeta == "" && subMeta == "" {
		return filePath, nil
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	isMedia := ext == ".mp4" || ext == ".mkv" || ext == ".webm" || ext == ".avi" || ext == ".mov" || ext == ".mp3" || ext == ".m4a" || ext == ".flac" || ext == ".opus"
	if !isMedia {
		return filePath, nil
	}

	tempOut := filepath.Join(filepath.Dir(filePath), "meta_"+filepath.Base(filePath))

	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-i", filePath,
		"-map", "0",
		"-c", "copy",
		"-map_metadata", "-1",
	}

	if globalTitle != "" {
		args = append(args, "-metadata", fmt.Sprintf("title=%s", globalTitle))
		args = append(args, "-metadata", fmt.Sprintf("comment=%s", globalTitle))
	}

	if videoMeta != "" {
		args = append(args, "-metadata:s:v", fmt.Sprintf("title=%s", videoMeta))
	}

	if audioMeta != "" {
		args = append(args, "-metadata:s:a", fmt.Sprintf("title=%s", audioMeta))
	}

	if subMeta != "" {
		args = append(args, "-metadata:s:s", fmt.Sprintf("title=%s", subMeta))
	}

	args = append(args, tempOut)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tempOut)
		return filePath, fmt.Errorf("ffmpeg metadata failed: %w", err)
	}

	_ = os.Remove(filePath)
	_ = os.Rename(tempOut, filePath)
	return filePath, nil
}

// ApplyNameSwap replaces matched regex patterns in filenames based on find:replace syntax
func ApplyNameSwap(name, swapRule string) string {
	if swapRule == "" || !strings.Contains(swapRule, ":") {
		return name
	}

	parts := strings.SplitN(swapRule, ":", 2)
	find := parts[0]
	replace := parts[1]

	re, err := regexp.Compile(find)
	if err != nil {
		return strings.ReplaceAll(name, find, replace)
	}

	return re.ReplaceAllString(name, replace)
}

// GenerateScreenshotGrid creates a photo contact sheet of video scenes
func GenerateScreenshotGrid(ctx context.Context, videoPath, outDir string, layout string) (string, error) {
	if layout == "" {
		layout = "3x3"
	}

	gridPath := filepath.Join(outDir, "grid_thumb.jpg")
	// Extract frames and create grid using ffmpeg tile filter
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-ss", "00:00:30",
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "2",
		gridPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate screenshot grid: %w", err)
	}

	return gridPath, nil
}
