package media

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type VideoMetadata struct {
	Duration float64 `json:"duration"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Codec    string  `json:"codec"`
}

// GetVideoMetadata probes video files for duration, dimensions, and codec
func GetVideoMetadata(ctx context.Context, videoPath string) (*VideoMetadata, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,codec_name,duration:format=duration",
		"-of", "json",
		videoPath,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var data struct {
		Streams []struct {
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			CodecName string `json:"codec_name"`
			Duration  string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(out, &data); err != nil {
		return nil, err
	}

	meta := &VideoMetadata{}
	if len(data.Streams) > 0 {
		meta.Width = data.Streams[0].Width
		meta.Height = data.Streams[0].Height
		meta.Codec = data.Streams[0].CodecName
		if d, err := strconv.ParseFloat(data.Streams[0].Duration, 64); err == nil && d > 0 {
			meta.Duration = d
		}
	}

	if meta.Duration == 0 {
		if d, err := strconv.ParseFloat(data.Format.Duration, 64); err == nil {
			meta.Duration = d
		}
	}

	return meta, nil
}

// GenerateThumbnail extracts a snapshot thumbnail from a video file using ffmpeg
func GenerateThumbnail(ctx context.Context, videoPath, outDir string) (string, error) {
	thumbPath := filepath.Join(outDir, "thumb.jpg")
	// Seek to 10 seconds or 10%
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-ss", "00:00:10",
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", "2",
		thumbPath,
	)

	if err := cmd.Run(); err != nil {
		// Fallback seek to 0s
		cmdFallback := exec.CommandContext(ctx, "ffmpeg",
			"-y",
			"-ss", "00:00:01",
			"-i", videoPath,
			"-vframes", "1",
			"-q:v", "2",
			thumbPath,
		)
		if errFb := cmdFallback.Run(); errFb != nil {
			return "", fmt.Errorf("failed to generate video thumbnail: %w", err)
		}
	}

	return thumbPath, nil
}

// GenerateSampleVideo cuts a 30-second sample clip from the media file
func GenerateSampleVideo(ctx context.Context, videoPath string) (string, error) {
	ext := filepath.Ext(videoPath)
	outPath := strings.TrimSuffix(videoPath, ext) + "_sample" + ext

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-ss", "00:01:00",
		"-i", videoPath,
		"-t", "30",
		"-c", "copy",
		outPath,
	)

	if err := cmd.Run(); err != nil {
		// Try re-encoding if copy fails
		cmdRecode := exec.CommandContext(ctx, "ffmpeg",
			"-y",
			"-ss", "00:00:10",
			"-i", videoPath,
			"-t", "30",
			outPath,
		)
		if errRe := cmdRecode.Run(); errRe != nil {
			return "", fmt.Errorf("sample video generation failed: %w", err)
		}
	}

	return outPath, nil
}

// SplitFile divides large files into chunks smaller than maxBytes (e.g. 2GB)
func SplitFile(srcPath string, maxBytes int64) ([]string, error) {
	return SplitFileCustom(srcPath, maxBytes, false)
}

// SplitFileCustom divides large files with optional equal chunk distribution
func SplitFileCustom(srcPath string, maxBytes int64, equalSplits bool) ([]string, error) {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return nil, err
	}

	totalSize := srcInfo.Size()
	if totalSize <= maxBytes {
		return []string{srcPath}, nil
	}

	chunkSize := maxBytes
	if equalSplits {
		numParts := (totalSize + maxBytes - 1) / maxBytes
		if numParts > 1 {
			chunkSize = (totalSize + numParts - 1) / numParts
		}
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	var parts []string
	partNum := 1
	buf := make([]byte, 4*1024*1024) // 4MB buffer

	for {
		partPath := fmt.Sprintf("%s.%03d", srcPath, partNum)
		partFile, err := os.Create(partPath)
		if err != nil {
			return nil, err
		}

		var written int64
		for written < chunkSize {
			toRead := int64(len(buf))
			if chunkSize-written < toRead {
				toRead = chunkSize - written
			}

			n, rErr := srcFile.Read(buf[:toRead])
			if n > 0 {
				if _, wErr := partFile.Write(buf[:n]); wErr != nil {
					partFile.Close()
					return nil, wErr
				}
				written += int64(n)
			}
			if rErr != nil {
				partFile.Close()
				if rErr == io.EOF {
					if written > 0 {
						parts = append(parts, partPath)
					}
					return parts, nil
				}
				return nil, rErr
			}
		}

		partFile.Close()
		parts = append(parts, partPath)
		partNum++
	}
}

// ExtractArchive extracts zip/7z/tar files safely into target directory
func ExtractArchive(ctx context.Context, archivePath, destDir string) error {
	// Try native 7z if available
	if _, err := exec.LookPath("7z"); err == nil {
		cmd := exec.CommandContext(ctx, "7z", "x", "-y", "-o"+destDir, archivePath)
		return cmd.Run()
	}

	// Native zip extractor fallback
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return unzipNative(archivePath, destDir)
	}

	return fmt.Errorf("extraction tool (7z) not found in system")
}

func unzipNative(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		cleanPath := filepath.Clean(f.Name)
		if strings.HasPrefix(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") || strings.HasPrefix(cleanPath, "\\") {
			continue // ZipSlip security guard
		}

		target := filepath.Join(dest, cleanPath)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateZip compresses files into a zip archive
func CreateZip(ctx context.Context, srcPath, zipPath string) error {
	outFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(filepath.Dir(srcPath), path)
		if err != nil {
			relPath = info.Name()
		}

		zipEntry, err := w.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipEntry, file)
		return err
	}

	return filepath.Walk(srcPath, walker)
}
