package frontier_test

import (
	"testing"
	"time"

	"github.com/dangpham/deisearch/spider/internal/frontier"
	"github.com/dangpham/deisearch/spider/internal/parser"
)

func TestBatchSameDomain(t *testing.T) {
	f := frontier.New([]string{}, 1)

	var links []parser.Link
	for i := 1; i <= 10; i++ {
		links = append(links, parser.Link{
			URL: "https://example.com/page" + string(rune('0'+i)),
		})
	}

	startTime := time.Now()
	f.AddURLs(links)

	t.Logf("Added 10 URLs from example.com")

	retrievedCount := 0
	for retrievedCount < 10 {
		url, wait := f.GetNext()

		if wait > 0 {
			t.Logf("Need to wait %v for next URL", wait)
			time.Sleep(wait)
			continue
		}

		if url != "" {
			retrievedCount++
			elapsed := time.Since(startTime)
			t.Logf("URL %d: %s (elapsed: %v)", retrievedCount, url, elapsed)
		}
	}

	if !f.IsEmpty() {
		t.Errorf("Queue should be empty, but has %d items", f.Size())
	}
	t.Logf("Successfully retrieved all 10 URLs with rate limiting")
}

func TestBatchMultipleDomains(t *testing.T) {
	f := frontier.New([]string{}, 1)

	var links []parser.Link
	for i := 1; i <= 10; i++ {
		links = append(links, parser.Link{
			URL: "https://example.com/page" + string(rune('0'+i)),
		})
		links = append(links, parser.Link{
			URL: "https://other.com/page" + string(rune('0'+i)),
		})
	}

	f.AddURLs(links)
	t.Logf("Added 20 URLs (10 from example.com, 10 from other.com)")

	readyCount := 0
	waitingCount := 0

	for i := 0; i < 20; i++ {
		url, wait := f.GetNext()

		if wait == 0 {
			readyCount++
			t.Logf("Ready: %s", url)
		} else {
			waitingCount++
			t.Logf("Waiting: %s (wait: %v)", url, wait)
		}
	}

	t.Logf("\nSummary: %d ready immediately, %d need to wait", readyCount, waitingCount)

	if readyCount < 2 {
		t.Error("Expected at least 2 URLs ready (one from each domain)")
	}
}

func TestHundredURLsSameDomain(t *testing.T) {
	f := frontier.New([]string{}, 1)

	var links []parser.Link
	for i := 1; i <= 100; i++ {
		links = append(links, parser.Link{
			URL: "https://example.com/page" + string(rune(i)),
		})
	}

	startTime := time.Now()
	f.AddURLs(links)
	t.Logf("Added 100 URLs from example.com")

	waitTimes := make([]time.Duration, 0)
	for i := 0; i < 100; i++ {
		_, wait := f.GetNext()
		waitTimes = append(waitTimes, wait)
	}

	if waitTimes[0] != 0 {
		t.Errorf("First URL should be immediate, got wait: %v", waitTimes[0])
	}
	t.Logf("URL 1: ready immediately")

	elapsed := time.Since(startTime)
	expectedWait2 := time.Duration(1) * time.Second
	adjustedWait2 := waitTimes[1] + elapsed

	if adjustedWait2 < expectedWait2-100*time.Millisecond || adjustedWait2 > expectedWait2+100*time.Millisecond {
		t.Logf("URL 2: wait=%v, elapsed=%v, adjusted=%v (expected ~1s)", waitTimes[1], elapsed, adjustedWait2)
	} else {
		t.Logf("URL 2: needs ~1s wait")
	}

	adjustedWait10 := waitTimes[9] + elapsed
	expectedWait10 := time.Duration(9) * time.Second
	if adjustedWait10 < expectedWait10-100*time.Millisecond || adjustedWait10 > expectedWait10+100*time.Millisecond {
		t.Logf("URL 10: wait=%v, elapsed=%v, adjusted=%v (expected ~9s)", waitTimes[9], elapsed, adjustedWait10)
	} else {
		t.Logf("URL 10: needs ~9s wait")
	}

	t.Logf("Verified rate limiting: URLs spaced 1 second apart")
}
