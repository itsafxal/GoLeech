package rss

import (
	"context"
	"encoding/xml"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
)

type RSSItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
	GUID  string `xml:"guid"`
}

type RSSFeed struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type Monitor struct {
	httpClient *http.Client
	stopCh     chan struct{}
}

func StartRSSMonitor(ctx context.Context) *Monitor {
	m := &Monitor{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		stopCh:     make(chan struct{}),
	}

	go m.loop(ctx)
	return m
}

func (m *Monitor) Stop() {
	close(m.stopCh)
}

func (m *Monitor) loop(ctx context.Context) {
	cfg := config.Get()
	if cfg.DisableRSS {
		return
	}

	delay := time.Duration(cfg.RssDelay) * time.Second
	if delay < 60*time.Second {
		delay = 60 * time.Second
	}

	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkFeeds(ctx)
		}
	}
}

func (m *Monitor) checkFeeds(ctx context.Context) {
	if db.GlobalStore == nil {
		return
	}

	feeds, err := db.GlobalStore.GetRSSFeeds()
	if err != nil {
		return
	}

	for _, feed := range feeds {
		m.processFeed(ctx, feed)
	}
}

func (m *Monitor) processFeed(ctx context.Context, f *db.RSSFeedRecord) {
	req, err := http.NewRequestWithContext(ctx, "GET", f.URL, nil)
	if err != nil {
		return
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var parsed RSSFeed
	if err := xml.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return
	}

	for _, item := range parsed.Channel.Items {
		itemID := item.GUID
		if itemID == "" {
			itemID = item.Link
		}
		if itemID == "" {
			itemID = item.Title
		}

		if db.GlobalStore.IsRSSItemSeen(f.ID, itemID) {
			continue
		}

		// Filter matching
		if f.FilterInclude != "" {
			if matched, _ := regexp.MatchString("(?i)"+f.FilterInclude, item.Title); !matched {
				continue
			}
		}
		if f.FilterExclude != "" {
			if matched, _ := regexp.MatchString("(?i)"+f.FilterExclude, item.Title); matched {
				continue
			}
		}

		// Dispatch automatic task
		tType := task.TypeMirror
		if strings.ToLower(f.Command) == "leech" {
			tType = task.TypeLeech
		}

		opts := &utils.TaskOptions{
			Link: item.Link,
			Name: item.Title,
		}

		t := task.NewTask(0, "RSS_Bot", f.ChatID, 0, tType, opts)
		task.GlobalManager.AddTask(t)

		_ = db.GlobalStore.MarkRSSItemSeen(f.ID, itemID)
	}
}
