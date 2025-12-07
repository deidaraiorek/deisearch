package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

type BrowserFetcher struct {
	userAgent string
}

func NewBrowserFetcher(userAgent string) *BrowserFetcher {
	return &BrowserFetcher{
		userAgent: userAgent,
	}
}

func (bf *BrowserFetcher) FetchHTML(ctx context.Context, urlStr string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(bf.userAgent),
		chromedp.Flag("disable-downloads", true),             // Prevent file downloads
		chromedp.Flag("disable-plugins", true),               // Disable plugins
		chromedp.Flag("disable-extensions", true),            // Disable extensions
		chromedp.Flag("disable-dev-shm-usage", true),         // Prevent memory issues
		chromedp.Flag("no-sandbox", false),                   // Enable sandbox for security
		chromedp.Flag("disable-web-security", false),         // Keep web security enabled
		chromedp.Flag("disable-background-networking", true), // Prevent background requests
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	var htmlContent string

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(urlStr),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return "", fmt.Errorf("browser fetch failed: %w", err)
	}

	return htmlContent, nil
}
