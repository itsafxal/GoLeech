package downloader

import (
	"regexp"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

type SourceType string

const (
	SourceTypeTelegramMedia SourceType = "Telegram Media"
	SourceTypeDirectURL     SourceType = "Direct URL"
	SourceTypeTorrent       SourceType = "Torrent"
	SourceTypeYTDLP         SourceType = "yt-dlp"
	SourceTypeGDrive        SourceType = "Google Drive"
	SourceTypeNone          SourceType = "None"
)

var (
	urlRegex    = regexp.MustCompile(`(?i)\bhttps?://[^\s<>"]+`)
	magnetRegex = regexp.MustCompile(`(?i)\bmagnet:\?[^\s<>"]+`)
)

// InputSource contains classified information about the download target
type InputSource struct {
	Type     SourceType
	Link     string
	IsMedia  bool
	MediaMsg *tg.NewMessage
}

// ClassifySource determines whether an incoming interaction represents a Telegram file,
// a direct HTTP/HTTPS URL, a Torrent/Magnet, or a streaming platform URL.
// It explicitly guards against Telegram's MessageMediaWebPage previews being misclassified as Telegram media.
func ClassifySource(m *tg.NewMessage, replyMsg *tg.NewMessage, explicitLink string) InputSource {
	cleanLink := strings.TrimSpace(explicitLink)

	// 1. If an explicit link was provided (via command args or flag)
	if cleanLink != "" {
		return classifyLink(cleanLink)
	}

	// 2. Check Replied Message
	if replyMsg != nil {
		// A. Check for real Telegram Document / Video / Audio / Voice files
		if doc := replyMsg.Document(); doc != nil {
			// Check if document is a .torrent file
			isTorrent := strings.EqualFold(doc.MimeType, "application/x-bittorrent")
			if !isTorrent {
				for _, attr := range doc.Attributes {
					if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
						if strings.HasSuffix(strings.ToLower(fn.FileName), ".torrent") {
							isTorrent = true
							break
						}
					}
				}
			}
			if isTorrent {
				return InputSource{
					Type:     SourceTypeTorrent,
					Link:     "tg://torrent",
					IsMedia:  true,
					MediaMsg: replyMsg,
				}
			}
			return InputSource{
				Type:     SourceTypeTelegramMedia,
				Link:     "tg://media",
				IsMedia:  true,
				MediaMsg: replyMsg,
			}
		}

		if replyMsg.Video() != nil || replyMsg.Audio() != nil || replyMsg.Voice() != nil || replyMsg.Animation() != nil {
			return InputSource{
				Type:     SourceTypeTelegramMedia,
				Link:     "tg://media",
				IsMedia:  true,
				MediaMsg: replyMsg,
			}
		}

		// Check for Telegram Photo (only if not a link preview)
		if replyMsg.Photo() != nil && replyMsg.WebPage() == nil {
			return InputSource{
				Type:     SourceTypeTelegramMedia,
				Link:     "tg://media",
				IsMedia:  true,
				MediaMsg: replyMsg,
			}
		}

		// B. Replied message has no real media: extract URL or magnet from text or webpage preview
		repText := replyMsg.Text()
		if magnet := magnetRegex.FindString(repText); magnet != "" {
			return InputSource{
				Type: SourceTypeTorrent,
				Link: magnet,
			}
		}
		if rawURL := urlRegex.FindString(repText); rawURL != "" {
			return classifyLink(rawURL)
		}

		if wp := replyMsg.WebPage(); wp != nil && wp.URL != "" {
			return classifyLink(wp.URL)
		}
	}

	// 3. Check Current Message itself (for auto-leech or direct inputs)
	if m != nil {
		if doc := m.Document(); doc != nil {
			isTorrent := strings.EqualFold(doc.MimeType, "application/x-bittorrent")
			if !isTorrent {
				for _, attr := range doc.Attributes {
					if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
						if strings.HasSuffix(strings.ToLower(fn.FileName), ".torrent") {
							isTorrent = true
							break
						}
					}
				}
			}
			if isTorrent {
				return InputSource{
					Type:     SourceTypeTorrent,
					Link:     "tg://torrent",
					IsMedia:  true,
					MediaMsg: m,
				}
			}
			return InputSource{
				Type:     SourceTypeTelegramMedia,
				Link:     "tg://media",
				IsMedia:  true,
				MediaMsg: m,
			}
		}

		if m.Video() != nil || m.Audio() != nil || m.Voice() != nil || m.Animation() != nil {
			return InputSource{
				Type:     SourceTypeTelegramMedia,
				Link:     "tg://media",
				IsMedia:  true,
				MediaMsg: m,
			}
		}

		msgText := m.Text()
		if magnet := magnetRegex.FindString(msgText); magnet != "" {
			return InputSource{
				Type: SourceTypeTorrent,
				Link: magnet,
			}
		}
		if rawURL := urlRegex.FindString(msgText); rawURL != "" {
			return classifyLink(rawURL)
		}

		if wp := m.WebPage(); wp != nil && wp.URL != "" {
			return classifyLink(wp.URL)
		}
	}

	return InputSource{Type: SourceTypeNone}
}

func classifyLink(link string) InputSource {
	linkLower := strings.ToLower(link)

	if strings.HasPrefix(linkLower, "magnet:?") || strings.HasSuffix(linkLower, ".torrent") {
		return InputSource{
			Type: SourceTypeTorrent,
			Link: link,
		}
	}

	if strings.Contains(linkLower, "youtube.com/") || strings.Contains(linkLower, "youtu.be/") ||
		strings.Contains(linkLower, "instagram.com/") || strings.Contains(linkLower, "tiktok.com/") ||
		strings.Contains(linkLower, "twitter.com/") || strings.Contains(linkLower, "x.com/") {
		return InputSource{
			Type: SourceTypeYTDLP,
			Link: link,
		}
	}

	if strings.Contains(linkLower, "drive.google.com") {
		return InputSource{
			Type: SourceTypeGDrive,
			Link: link,
		}
	}

	if strings.HasPrefix(linkLower, "http://") || strings.HasPrefix(linkLower, "https://") {
		return InputSource{
			Type: SourceTypeDirectURL,
			Link: link,
		}
	}

	return InputSource{
		Type: SourceTypeDirectURL,
		Link: link,
	}
}
