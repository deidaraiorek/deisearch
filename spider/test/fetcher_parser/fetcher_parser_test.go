package parser_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dangpham/deisearch/spider/internal/fetcher"
	"github.com/dangpham/deisearch/spider/internal/parser"
)

func TestParserIntegration(t *testing.T) {
	f := fetcher.New("TestBot/1.0")
	p := parser.New()

	testURL := "https://golang.org"

	t.Logf("Testing with: %s\n", testURL)

	resp, err := f.Fetch(context.Background(), testURL)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}

	t.Logf("Fetched! Status: %d", resp.StatusCode)

	page, links, err := p.Parse(resp, testURL)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	fmt.Println("\nPage Info:")
	fmt.Printf("  Title: %s\n", page.Title)
	fmt.Printf("  Description: %s\n", page.Description)
	fmt.Printf("  Content length: %d characters\n", len(page.Content))
	fmt.Printf("  Links found: %d\n", len(links))

	if len(page.Content) > 200 {
		fmt.Println("\nSample Content Preview:", page.Content)
	} else {
		fmt.Println("\nSample Content Preview:", page.Content)
	}

	fmt.Println("\nFirst 5 links:")
	for i, link := range links {
		if i >= 5 {
			break
		}
		fmt.Printf("  [%d] %s\n", i+1, link.URL)
	}
}
