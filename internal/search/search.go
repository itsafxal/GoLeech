package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type SearchResult struct {
	Name     string `json:"name"`
	Size     string `json:"size"`
	Seeders  int    `json:"seeders"`
	Leechers int    `json:"leechers"`
	Magnet   string `json:"magnet"`
	Engine   string `json:"engine"`
}

type Provider interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

type APISearchProvider struct {
	apiBaseURL string
	client     *http.Client
}

func NewAPISearchProvider(apiBaseURL string) *APISearchProvider {
	if apiBaseURL == "" {
		apiBaseURL = "https://torrents-api.ryukme.repl.co"
	}
	return &APISearchProvider{
		apiBaseURL: apiBaseURL,
		client:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *APISearchProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	reqURL := fmt.Sprintf("%s/all/%s", p.apiBaseURL, url.PathEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResults []struct {
		Name     string `json:"name"`
		Size     string `json:"size"`
		Seeders  string `json:"seeders"`
		Leechers string `json:"leechers"`
		Magnet   string `json:"magnet"`
		Site     string `json:"site"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResults); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, r := range apiResults {
		var s, l int
		fmt.Sscanf(r.Seeders, "%d", &s)
		fmt.Sscanf(r.Leechers, "%d", &l)

		results = append(results, SearchResult{
			Name:     r.Name,
			Size:     r.Size,
			Seeders:  s,
			Leechers: l,
			Magnet:   r.Magnet,
			Engine:   r.Site,
		})
	}

	return results, nil
}
