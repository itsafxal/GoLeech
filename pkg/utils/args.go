package utils

import (
	"strconv"
	"strings"
)

// TaskOptions holds flags parsed from commands like /mirror and /leech
type TaskOptions struct {
	Link          string
	Name          string
	AsDocument    bool
	AsMedia       bool
	Extract       bool
	Zip           bool
	SampleVideo   bool
	Screenshot    bool
	ForceStart    bool
	SelectFiles   bool
	Bulk          bool
	MultiCount    int
	SplitSize     int64
	UploadDest    string // Cloud destination path or GDrive ID or Rclone remote:path
	DumpChat      string // Leech dump channel ID or name
	CustomThumb   string
	AudioCaption  string
	VideoCaption  string
	CustomHeader  string
	AuthUser      string
	AuthPass      string
	Subfolder     string // -m <folder>
	ConvertAudio  string // -ca <mp3/aac/flac>
	ConvertVideo  string // -cv <mp4/mkv/webm>
	NameSwap      string // -ns <find:replace>
	MetadataTitle string // -meta <title>
	FFmpegArgs    string // -ff <custom args>
	TrimLength    string // -tl <duration>
	DownloadOnly  bool   // -d
	JoinParts     bool   // -j
	AudioOnly     bool   // -ad (extract audio from video)
	RcloneFlags   string // -rcf <custom rclone flags>
	SeedrToken    string // -seedr <token>
	ExtraArgs     map[string]string
}

// ParseTaskOptions parses command string tokens into structured TaskOptions
func ParseTaskOptions(input string) *TaskOptions {
	opts := &TaskOptions{
		MultiCount: 1,
		ExtraArgs:  make(map[string]string),
	}

	tokens := tokenizeInput(input)
	if len(tokens) == 0 {
		return opts
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		switch token {
		case "-doc":
			opts.AsDocument = true
		case "-med":
			opts.AsMedia = true
		case "-e":
			opts.Extract = true
		case "-z":
			opts.Zip = true
		case "-sv":
			opts.SampleVideo = true
		case "-ss":
			opts.Screenshot = true
		case "-f":
			opts.ForceStart = true
		case "-s":
			opts.SelectFiles = true
		case "-b":
			opts.Bulk = true
		case "-n":
			if i+1 < len(tokens) {
				opts.Name = tokens[i+1]
				i++
			}
		case "-up":
			if i+1 < len(tokens) {
				opts.UploadDest = tokens[i+1]
				i++
			}
		case "-ud":
			if i+1 < len(tokens) {
				opts.DumpChat = tokens[i+1]
				i++
			}
		case "-t":
			if i+1 < len(tokens) {
				opts.CustomThumb = tokens[i+1]
				i++
			}
		case "-i":
			if i+1 < len(tokens) {
				if count, err := strconv.Atoi(tokens[i+1]); err == nil && count > 0 {
					opts.MultiCount = count
				}
				i++
			}
		case "-sp":
			if i+1 < len(tokens) {
				opts.SplitSize = parseSizeToBytes(tokens[i+1])
				i++
			}
		case "-au":
			if i+1 < len(tokens) {
				opts.AuthUser = tokens[i+1]
				i++
			}
		case "-ap":
			if i+1 < len(tokens) {
				opts.AuthPass = tokens[i+1]
				i++
			}
		case "-d":
			opts.DownloadOnly = true
		case "-j":
			opts.JoinParts = true
		case "-ad":
			opts.AudioOnly = true
		case "-m":
			if i+1 < len(tokens) {
				opts.Subfolder = tokens[i+1]
				i++
			}
		case "-ca":
			if i+1 < len(tokens) {
				opts.ConvertAudio = tokens[i+1]
				i++
			}
		case "-cv":
			if i+1 < len(tokens) {
				opts.ConvertVideo = tokens[i+1]
				i++
			}
		case "-ns":
			if i+1 < len(tokens) {
				opts.NameSwap = tokens[i+1]
				i++
			}
		case "-meta":
			if i+1 < len(tokens) {
				opts.MetadataTitle = tokens[i+1]
				i++
			}
		case "-ff":
			if i+1 < len(tokens) {
				opts.FFmpegArgs = tokens[i+1]
				i++
			}
		case "-tl":
			if i+1 < len(tokens) {
				opts.TrimLength = tokens[i+1]
				i++
			}
		case "-rcf":
			if i+1 < len(tokens) {
				opts.RcloneFlags = tokens[i+1]
				i++
			}
		case "-seedr":
			if i+1 < len(tokens) {
				opts.SeedrToken = tokens[i+1]
				i++
			}
		case "-h":
			if i+1 < len(tokens) {
				opts.CustomHeader = tokens[i+1]
				i++
			}
		default:
			if strings.HasPrefix(token, "-") {
				if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
					opts.ExtraArgs[strings.TrimPrefix(token, "-")] = tokens[i+1]
					i++
				} else {
					opts.ExtraArgs[strings.TrimPrefix(token, "-")] = "true"
				}
			} else if opts.Link == "" {
				opts.Link = token
			}
		}
	}

	return opts
}

func tokenizeInput(s string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range s {
		switch {
		case inQuotes:
			if ch == quoteChar {
				inQuotes = false
			} else {
				current.WriteRune(ch)
			}
		case ch == '"' || ch == '\'':
			inQuotes = true
			quoteChar = ch
		case ch == ' ' || ch == '\t' || ch == '\n':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func parseSizeToBytes(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	multi := int64(1)
	if strings.HasSuffix(s, "G") || strings.HasSuffix(s, "GB") {
		multi = 1024 * 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "GB"), "G")
	} else if strings.HasSuffix(s, "M") || strings.HasSuffix(s, "MB") {
		multi = 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "MB"), "M")
	} else if strings.HasSuffix(s, "K") || strings.HasSuffix(s, "KB") {
		multi = 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "KB"), "K")
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(val * float64(multi))
}
