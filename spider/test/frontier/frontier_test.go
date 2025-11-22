package frontier_test

import (
	"testing"
	"time"

	"github.com/dangpham/deisearch/spider/internal/frontier"
	"github.com/dangpham/deisearch/spider/internal/parser"
)

func TestFrontier(t *testing.T) {
	crawledURLs := []string{
		"https://golang.org",
		"https://golang.org/doc",
	}

	f := frontier.New(crawledURLs, 1)

	t.Logf("Frontier created with %d seen URLs", len(crawledURLs))

	f.AddURL("https://golang.org")
	if f.Size() != 0 {
		t.Error("Already crawled URL should not be added to queue")
	}
	t.Log("Deduplication works - already crawled URL ignored")

	f.AddURL("https://go.dev")
	if f.Size() != 1 {
		t.Errorf("Expected queue size 1, got %d", f.Size())
	}
	t.Log("New URL added to queue")

	url, waitTime := f.GetNext()
	if url != "https://go.dev" {
		t.Errorf("Expected https://go.dev, got %s", url)
	}
	if waitTime != 0 {
		t.Errorf("Expected wait time 0, got %v", waitTime)
	}
	t.Logf("Retrieved URL: %s (wait: %v)", url, waitTime)

	links := []parser.Link{
		{URL: "https://example.com/page1"},
		{URL: "https://example.com/page2"},
		{URL: "https://example.com/page3"},
		{URL: "https://other.com/page1"},
	}

	f.AddURLs(links)
	t.Logf("Added %d links, queue size: %d", len(links), f.Size())

	url1, wait1 := f.GetNext()
	t.Logf("First URL: %s (wait: %v)", url1, wait1)

	url2, wait2 := f.GetNext()
	t.Logf("Second URL: %s (wait: %v)", url2, wait2)

	if wait2 > 0 {
		t.Logf("Rate limiting active - need to wait %v", wait2)
		time.Sleep(wait2)
		url2, wait2 = f.GetNext()
		t.Logf("After waiting: %s (wait: %v)", url2, wait2)
	}

	duplicateLinks := []parser.Link{
		{URL: "https://example.com/page1"},
		{URL: "https://newsite.com/page1"},
	}

	sizeBefore := f.Size()
	f.AddURLs(duplicateLinks)
	sizeAfter := f.Size()

	if sizeAfter-sizeBefore != 1 {
		t.Errorf("Expected 1 new URL, got %d", sizeAfter-sizeBefore)
	}
	t.Log("Duplicate URLs filtered in batch add")

	for !f.IsEmpty() {
		url, wait := f.GetNext()
		if wait > 0 {
			time.Sleep(wait)
			url, _ = f.GetNext()
		}
		if url != "" {
			t.Logf("Drained: %s", url)
		}
	}

	if !f.IsEmpty() {
		t.Error("Queue should be empty")
	}
	t.Log("Queue emptied successfully")
}

func TestFrontierPriority(t *testing.T) {
	f := frontier.New([]string{}, 2)

	links := []parser.Link{
		{URL: "https://example.com/1"},
		{URL: "https://example.com/2"},
		{URL: "https://example.com/3"},
	}

	f.AddURLs(links)

	url1, wait1 := f.GetNext()
	if wait1 != 0 {
		t.Errorf("First URL should be immediate, got wait: %v", wait1)
	}
	t.Logf("URL 1: %s (immediate)", url1)

	url2, wait2 := f.GetNext()
	if url2 != "" {
		t.Error("Second URL should not be ready yet")
	}
	if wait2 == 0 {
		t.Error("Should need to wait for second URL")
	}
	t.Logf("Priority queue working - next URL needs to wait: %v", wait2)
}
