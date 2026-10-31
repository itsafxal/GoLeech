package ytdlp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
)

type Downloader struct{}

func NewDownloader() *Downloader {
	return &Downloader{}
}

var (
	progressRe = regexp.MustCompile(`\[download\]\s+([\d\.]+)%\s+of\s+~?([\d\.]+[A-Za-z]+)\s+at\s+([\d\.]+[A-Za-z]+/s)\s+ETA\s+([\d:]+)`)
	filenameRe = regexp.MustCompile(`\[download\]\s+Destination:\s+(.+)`)
	mergerRe   = regexp.MustCompile(`\[Merger\]\s+Merging\s+formats\s+into\s+"(.+)"`)
)

func (d *Downloader) Download(ctx context.Context, t *task.Task, onProgress downloader.ProgressCallback) (string, error) {
	cfg := config.Get()
	taskDir := filepath.Join(cfg.DownloadDir, t.GID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create ytdlp download directory: %w", err)
	}
	t.LocalPath = taskDir

	outTemplate := filepath.Join(taskDir, "%(title)s.%(ext)s")
	if t.Options.Name != "" {
		outTemplate = filepath.Join(taskDir, utils.CleanFilename(t.Options.Name)+".%(ext)s")
	}

	args := []string{
		"--newline",
		"--no-playlist",
		"--concurrent-fragments", "4",
		"-o", outTemplate,
		t.Options.Link,
	}

	// Check if user requested audio only or specific quality
	if t.Options.AsDocument {
		args = append([]string{"-f", "bestvideo+bestaudio/best"}, args...)
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to open yt-dlp stdout: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start yt-dlp: %w (ensure yt-dlp is installed and in PATH)", err)
	}

	scanner := bufio.NewScanner(stdout)
	var downloadedFile string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Detect destination filename
		if m := filenameRe.FindStringSubmatch(line); len(m) > 1 {
			downloadedFile = strings.TrimSpace(m[1])
			t.SetName(filepath.Base(downloadedFile))
		}
		if m := mergerRe.FindStringSubmatch(line); len(m) > 1 {
			downloadedFile = strings.TrimSpace(m[1])
			t.SetName(filepath.Base(downloadedFile))
		}

		// Parse progress line: [download]  45.2% of 120.50MiB at 4.25MiB/s ETA 00:15
		if m := progressRe.FindStringSubmatch(line); len(m) > 4 {
			pct, _ := strconv.ParseFloat(m[1], 64)
			speedBytes := parseYtSpeed(m[3])
			totalBytes := parseYtSize(m[2])
			curBytes := int64((pct / 100.0) * float64(totalBytes))

			if onProgress != nil {
				onProgress(curBytes, totalBytes, speedBytes)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("yt-dlp execution error: %w", err)
	}

	// If downloadedFile wasn't matched directly, scan directory for the downloaded file
	if downloadedFile == "" || !fileExists(downloadedFile) {
		files, _ := os.ReadDir(taskDir)
		for _, f := range files {
			if !f.IsDir() && !strings.HasSuffix(f.Name(), ".part") && !strings.HasSuffix(f.Name(), ".ytdl") {
				downloadedFile = filepath.Join(taskDir, f.Name())
				t.SetName(f.Name())
				break
			}
		}
	}

	if downloadedFile == "" {
		return "", fmt.Errorf("yt-dlp completed but output file was not found")
	}

	return downloadedFile, nil
}

func parseYtSpeed(s string) float64 {
	s = strings.TrimSuffix(strings.ToUpper(s), "/S")
	multi := 1.0
	if strings.HasSuffix(s, "MIB") || strings.HasSuffix(s, "MB") {
		multi = 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "MIB"), "MB")
	} else if strings.HasSuffix(s, "KIB") || strings.HasSuffix(s, "KB") {
		multi = 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "KIB"), "KB")
	} else if strings.HasSuffix(s, "GIB") || strings.HasSuffix(s, "GB") {
		multi = 1024 * 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "GIB"), "GB")
	}
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return val * multi
}

func parseYtSize(s string) int64 {
	s = strings.ToUpper(s)
	multi := int64(1)
	if strings.HasSuffix(s, "MIB") || strings.HasSuffix(s, "MB") {
		multi = 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "MIB"), "MB")
	} else if strings.HasSuffix(s, "GIB") || strings.HasSuffix(s, "GB") {
		multi = 1024 * 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "GIB"), "GB")
	} else if strings.HasSuffix(s, "KIB") || strings.HasSuffix(s, "KB") {
		multi = 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "KIB"), "KB")
	}
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return int64(val * float64(multi))
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
