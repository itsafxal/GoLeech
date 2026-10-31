package providers

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/task"
)

type RcloneUploader struct{}

func NewRcloneUploader() *RcloneUploader {
	return &RcloneUploader{}
}

var (
	rcloneProgressRe = regexp.MustCompile(`Transferred:\s+([\d\.]+\s+[A-Za-z]+)\s+/\s+([\d\.]+\s+[A-Za-z]+),\s+([\d\.]+)%,\s+([\d\.]+\s+[A-Za-z]+/s),\s+ETA\s+([\w\d]+)`)
)

func (u *RcloneUploader) Upload(ctx context.Context, t *task.Task, filePath string, onProgress downloader.ProgressCallback) (*UploadResult, error) {
	cfg := config.Get()
	remoteDest := t.Options.UploadDest
	if remoteDest == "" {
		remoteDest = cfg.RclonePath
	}
	if remoteDest == "" {
		return nil, fmt.Errorf("no rclone destination specified (-up or RCLONE_PATH)")
	}

	args := []string{
		"copy",
		filePath,
		remoteDest,
		"--progress",
		"--stats", "1s",
	}

	if cfg.RcloneFlags != "" {
		extra := strings.Fields(cfg.RcloneFlags)
		args = append(args, extra...)
	}

	cmd := exec.CommandContext(ctx, "rclone", args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe rclone stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start rclone: %w (ensure rclone binary is installed)", err)
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if m := rcloneProgressRe.FindStringSubmatch(line); len(m) > 4 {
			pct, _ := strconv.ParseFloat(m[3], 64)
			curBytes := parseRcloneBytes(m[1])
			totBytes := parseRcloneBytes(m[2])
			spdBytes := parseRcloneBytes(strings.TrimSuffix(m[4], "/s"))

			if onProgress != nil {
				onProgress(curBytes, totBytes, float64(spdBytes))
			}
			_ = pct
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("rclone upload error: %w", err)
	}

	return &UploadResult{
		URL: remoteDest,
	}, nil
}

// Clone performs direct cloud-to-cloud transfers between remotes without local disk usage
func (u *RcloneUploader) Clone(ctx context.Context, t *task.Task, src, dst string, onProgress downloader.ProgressCallback) (*UploadResult, error) {
	cfg := config.Get()
	args := []string{
		"copy",
		src,
		dst,
		"--progress",
		"--stats", "1s",
	}

	if cfg.RcloneFlags != "" {
		extra := strings.Fields(cfg.RcloneFlags)
		args = append(args, extra...)
	}

	cmd := exec.CommandContext(ctx, "rclone", args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe rclone stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start rclone: %w (ensure rclone binary is installed)", err)
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if m := rcloneProgressRe.FindStringSubmatch(line); len(m) > 4 {
			curBytes := parseRcloneBytes(m[1])
			totBytes := parseRcloneBytes(m[2])
			spdBytes := parseRcloneBytes(strings.TrimSuffix(m[4], "/s"))

			if onProgress != nil {
				onProgress(curBytes, totBytes, float64(spdBytes))
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("rclone clone error: %w", err)
	}

	return &UploadResult{
		URL: dst,
	}, nil
}

func parseRcloneBytes(s string) int64 {
	parts := strings.Fields(strings.ToUpper(s))
	if len(parts) < 2 {
		return 0
	}
	val, _ := strconv.ParseFloat(parts[0], 64)
	multi := int64(1)
	switch parts[1] {
	case "GIB", "GB":
		multi = 1024 * 1024 * 1024
	case "MIB", "MB":
		multi = 1024 * 1024
	case "KIB", "KB":
		multi = 1024
	case "B", "BYTES":
		multi = 1
	}
	return int64(val * float64(multi))
}
