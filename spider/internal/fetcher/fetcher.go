package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

type Fetcher struct {
	client      *http.Client
	robotsCache map[string]*robotstxt.RobotsData
	robotsMu    sync.RWMutex
	userAgent   string
}

func New(userAgent string) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		robotsCache: make(map[string]*robotstxt.RobotsData),
		userAgent:   userAgent,
	}
}

func (f *Fetcher) Fetch(ctx context.Context, urlStr string) (*http.Response, error) {
	if !f.IsAllowed(urlStr) {
		return nil, fmt.Errorf("disallowed by robots.txt")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}

	return resp, nil
}

func (f *Fetcher) IsAllowed(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", u.Scheme, u.Host)

	f.robotsMu.RLock()
	robots, exists := f.robotsCache[robotsURL]
	f.robotsMu.RUnlock()

	if !exists {
		robots = f.fetchRobotsTxt(robotsURL)
		f.robotsMu.Lock()
		f.robotsCache[robotsURL] = robots
		f.robotsMu.Unlock()
	}

	if robots == nil {
		return true
	}

	group := robots.FindGroup(f.userAgent)
	return group.Test(u.Path)
}

func (f *Fetcher) fetchRobotsTxt(robotsURL string) *robotstxt.RobotsData {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	robots, err := robotstxt.FromResponse(resp)
	if err != nil {
		return nil
	}
	return robots
}
