package storage_test

import (
	"os"
	"testing"
	"time"

	"github.com/dangpham/deisearch/spider/internal/storage"
)

func TestDatabase(t *testing.T) {
	dbPath := "./test_search.db"
	defer os.Remove(dbPath)

	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	t.Log("Database created successfully")

	page := &storage.Page{
		URL:         "https://golang.org",
		Title:       "The Go Programming Language",
		Description: "Go is an open source programming language",
		Content:     "Build simple, secure, scalable systems with Go",
		StatusCode:  200,
		CrawledAt:   time.Now(),
	}

	err = db.SavePage(page)
	if err != nil {
		t.Fatalf("Failed to save page: %v", err)
	}
	t.Log("Page saved successfully")

	retrieved, err := db.GetPage("https://golang.org")
	if err != nil {
		t.Fatalf("Failed to get page: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Page not found")
	}

	if retrieved.Title != page.Title {
		t.Errorf("Expected title '%s', got '%s'", page.Title, retrieved.Title)
	}
	t.Logf("Page retrieved: %s\n%s", retrieved.Title, page.Content)

	crawledURLs, err := db.LoadAllCrawledURLs()
	if err != nil {
		t.Fatalf("LoadAllCrawledURLs error: %v", err)
	}

	found := false
	for _, url := range crawledURLs {
		if url == "https://golang.org" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("URL should be in crawled URLs list")
	}

	links := []string{"https://go.dev", "https://golang.org/doc", "https://pkg.go.dev"}
	err = db.SaveLinks("https://golang.org", links)
	if err != nil {
		t.Fatalf("SaveLinks error: %v", err)
	}
	t.Logf("Saved %d links from golang.org", len(links))

	count, _ := db.GetPageCount()
	t.Logf("Total pages crawled: %d", count)
}
