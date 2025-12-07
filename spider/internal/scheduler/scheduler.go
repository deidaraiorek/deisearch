package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dangpham/deisearch/spider/internal/fetcher"
	"github.com/dangpham/deisearch/spider/internal/frontier"
	"github.com/dangpham/deisearch/spider/internal/parser"
	"github.com/dangpham/deisearch/spider/internal/storage"
)

type Config struct {
	Workers      int
	RateLimitSec float32
	MaxPages     int
	MaxDepth     int
	UserAgent    string
}

type Scheduler struct {
	config         *Config
	frontier       *frontier.Frontier
	fetcher        *fetcher.Fetcher
	browserFetcher *fetcher.BrowserFetcher
	parser         *parser.Parser
	db             *storage.Database

	pageCount           int
	browserFetchedCount int
	mu                  sync.Mutex
}

func New(db *storage.Database, config *Config) *Scheduler {
	if config.UserAgent == "" {
		config.UserAgent = "DeiSearchBot/1.0"
	}
	if config.RateLimitSec == 0 {
		config.RateLimitSec = 1
	}
	if config.Workers == 0 {
		config.Workers = 20
	}

	crawledURLs, err := db.LoadAllCrawledURLs()
	if err != nil {
		log.Printf("Warning: Failed to load crawled URLs: %v", err)
		crawledURLs = []string{}
	}

	return &Scheduler{
		config:         config,
		frontier:       frontier.New(crawledURLs, config.RateLimitSec),
		fetcher:        fetcher.New(config.UserAgent),
		browserFetcher: fetcher.NewBrowserFetcher(config.UserAgent),
		parser:         parser.New(),
		db:             db,
	}
}

func (s *Scheduler) AddSeed(url string) error {
	s.frontier.AddURL(url)
	return nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	log.Printf("Starting crawler with %d workers", s.config.Workers)

	var wg sync.WaitGroup
	for i := 0; i < s.config.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.worker(ctx, workerID)
		}(i)
	}

	wg.Wait()

	s.mu.Lock()
	browserCount := s.browserFetchedCount
	s.mu.Unlock()

	log.Printf("Crawling completed. Total pages: %d (browser-fetched: %d)", s.pageCount, browserCount)
	return nil
}

func (s *Scheduler) worker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", workerID)
			return
		default:
		}

		if s.shouldStop() {
			log.Printf("Worker %d: Reached max pages limit", workerID)
			return
		}

		url, wait := s.frontier.GetNext()

		if url == "" {
			if wait > 0 {
				time.Sleep(wait)
				continue
			}

			if s.frontier.IsEmpty() {
				log.Printf("Worker %d: Frontier empty, exiting", workerID)
				return
			}

			time.Sleep(100 * time.Millisecond)
			continue
		}

		log.Printf("Worker %d: Crawling %s", workerID, url)

		crawled, err := s.crawlURL(ctx, url)
		if err != nil {
			log.Printf("Worker %d: Error crawling %s: %v", workerID, url, err)
		}

		if crawled {
			s.incrementPageCount()
		}
	}
}

func (s *Scheduler) crawlURL(ctx context.Context, url string) (bool, error) {
	// Phase 1: Try with fast HTTP fetcher
	resp, err := s.fetcher.Fetch(ctx, url)
	if err != nil {
		return false, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("non-200 status: %d", resp.StatusCode)
	}

	// Security: Validate Content-Type to prevent processing non-HTML files
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !isHTMLContentType(contentType) {
		log.Printf("🔒 Skipping non-HTML content type: %s for %s", contentType, url)
		return false, nil
	}

	// Security: Check content length to prevent huge downloads
	if resp.ContentLength > 10*1024*1024 { // 10MB limit
		log.Printf("🔒 Skipping oversized content (%d bytes) for %s", resp.ContentLength, url)
		return false, nil
	}

	page, links, err := s.parser.Parse(resp, url)
	if err != nil {
		return false, fmt.Errorf("🔴 parse failed: %w", err)
	}

	if page == nil {
		log.Printf("Skipping non-English page: %s", url)
		return false, nil
	}

	// Phase 2: If content is insufficient, retry with browser
	if !page.HasSufficientContent() {
		log.Printf("⚠️  Insufficient content from HTTP fetch, retrying with browser: %s", url)

		htmlContent, err := s.browserFetcher.FetchHTML(ctx, url)
		if err != nil {
			log.Printf("⚠️  Browser fetch failed, skipping page: %v", err)
			return false, nil
		}

		// Parse the browser-fetched HTML
		browserPage, browserLinks, err := s.parser.ParseHTML(htmlContent, url)
		if err != nil {
			log.Printf("⚠️  Browser parse failed, skipping page: %v", err)
			return false, nil
		}

		if browserPage.HasSufficientContent() {
			log.Printf("✅ Browser fetch successful for: %s", url)
			page = browserPage
			links = browserLinks
			s.incrementBrowserFetchedCount()
		} else {
			// Both HTTP and browser fetch failed to get sufficient content
			log.Printf("❌ Skipping page with insufficient content (even after browser fetch): %s", url)
			return false, nil
		}
	}

	normalizedURL := parser.NormalizeURLString(page.URL)
	dbPage := &storage.Page{
		URL:         normalizedURL,
		Title:       page.Title,
		Description: page.Description,
		Content:     page.Content,
		StatusCode:  page.StatusCode,
		CrawledAt:   time.Now(),
	}

	if err := s.db.SavePage(dbPage); err != nil {
		return false, fmt.Errorf("🔴 save page failed: %w", err)
	}

	linkURLs := make([]string, len(links))
	for i, link := range links {
		linkURLs[i] = link.URL
	}

	if len(linkURLs) > 0 {
		if err := s.db.SaveLinks(normalizedURL, linkURLs); err != nil {
			log.Printf("🔴 Warning: Failed to save links: %v", err)
		}

		s.frontier.AddURLs(links)
		log.Printf("Worker: Added %d new links to frontier", len(links))
	}

	return true, nil
}

func (s *Scheduler) incrementBrowserFetchedCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.browserFetchedCount++
}

func (s *Scheduler) shouldStop() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.MaxPages > 0 && s.pageCount >= s.config.MaxPages {
		return true
	}
	return false
}

func (s *Scheduler) incrementPageCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pageCount++
}

func (s *Scheduler) GetStats() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]interface{}{
		"pages_crawled": s.pageCount,
		"queue_size":    s.frontier.Size(),
	}
}

func isHTMLContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	htmlTypes := []string{
		"text/html",
		"application/xhtml+xml",
		"application/xhtml",
	}

	for _, htmlType := range htmlTypes {
		if strings.HasPrefix(contentType, htmlType) {
			return true
		}
	}

	return false
}
