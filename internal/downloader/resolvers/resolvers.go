package resolvers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	mediafireRe   = regexp.MustCompile(`https?://(?:www\.)?mediafire\.com/(?:file|download)/([a-zA-Z0-9]+)`)
	pixeldrainRe  = regexp.MustCompile(`https?://(?:www\.)?pixeldrain\.com/u/([a-zA-Z0-9]+)`)
	buzzheavierRe = regexp.MustCompile(`https?://(?:www\.)?buzzheavier\.com/f/([a-zA-Z0-9_-]+)`)
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// ResolveLink attempts to parse any known hosting service and return the direct raw URL
func ResolveLink(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, nil
	}

	host := strings.ToLower(parsed.Host)

	// Pixeldrain
	if strings.Contains(host, "pixeldrain.com") {
		if m := pixeldrainRe.FindStringSubmatch(rawURL); len(m) > 1 {
			return fmt.Sprintf("https://pixeldrain.com/api/file/%s?download", m[1]), nil
		}
	}

	// Mediafire
	if strings.Contains(host, "mediafire.com") {
		if direct, err := resolveMediafire(rawURL); err == nil && direct != "" {
			return direct, nil
		}
	}

	// Buzzheavier
	if strings.Contains(host, "buzzheavier.com") {
		if direct, err := resolveBuzzheavier(rawURL); err == nil && direct != "" {
			return direct, nil
		}
	}

	// Default fallback to raw URL
	return rawURL, nil
}

func resolveMediafire(rawURL string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Look for download button link
	re := regexp.MustCompile(`href="(https?://download\d+\.mediafire\.com/[^"]+)"`)
	m := re.FindStringSubmatch(string(body))
	if len(m) > 1 {
		return m[1], nil
	}
	return "", fmt.Errorf("mediafire download link not found")
}

func resolveBuzzheavier(rawURL string) (string, error) {
	// Buzzheavier direct download API / download path
	if strings.HasSuffix(rawURL, "/download") {
		return rawURL, nil
	}
	return rawURL + "/download", nil
}
