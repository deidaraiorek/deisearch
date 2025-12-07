package frontier

import (
	"container/heap"
	"sync"
	"time"

	"github.com/dangpham/deisearch/spider/internal/parser"
)

type URLItem struct {
	URL         string
	AvailableAt time.Time
	index       int
}

type PriorityQueue []*URLItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].AvailableAt.Before(pq[j].AvailableAt)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*URLItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

type Frontier struct {
	queue         *PriorityQueue
	seen          map[string]bool
	lastCrawlTime map[string]time.Time
	mu            sync.Mutex
	rateLimit     time.Duration
}

func New(crawledURLs []string, rateLimitSeconds float32) *Frontier {
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	seen := make(map[string]bool)
	for _, url := range crawledURLs {
		normalizedURL := parser.NormalizeURLString(url)
		seen[normalizedURL] = true
	}

	return &Frontier{
		queue:         &pq,
		seen:          seen,
		lastCrawlTime: make(map[string]time.Time),
		rateLimit:     time.Duration(rateLimitSeconds * float32(time.Second)),
	}
}

func (f *Frontier) AddURL(url string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	normalizedURL := parser.NormalizeURLString(url)

	if f.seen[normalizedURL] {
		return
	}

	f.seen[normalizedURL] = true

	item := &URLItem{
		URL:         normalizedURL,
		AvailableAt: time.Now(),
	}
	heap.Push(f.queue, item)
}

func (f *Frontier) AddURLs(links []parser.Link) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()

	for _, link := range links {
		if f.seen[link.URL] {
			continue
		}

		f.seen[link.URL] = true

		domain := parser.ExtractDomain(link.URL)

		availableAt := now
		if lastScheduled, exists := f.lastCrawlTime[domain]; exists {
			if lastScheduled.After(now) {
				availableAt = lastScheduled
			}
		}

		f.lastCrawlTime[domain] = availableAt.Add(f.rateLimit)

		item := &URLItem{
			URL:         link.URL,
			AvailableAt: availableAt,
		}
		heap.Push(f.queue, item)
	}
}

func (f *Frontier) GetNext() (string, time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.queue.Len() == 0 {
		return "", 0
	}

	item := (*f.queue)[0]

	now := time.Now()
	if item.AvailableAt.After(now) {
		return "", item.AvailableAt.Sub(now)
	}

	item = heap.Pop(f.queue).(*URLItem)

	domain := parser.ExtractDomain(item.URL)
	f.lastCrawlTime[domain] = now
	return item.URL, 0
}

func (f *Frontier) Size() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queue.Len()
}

func (f *Frontier) IsEmpty() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queue.Len() == 0
}

func (f *Frontier) HasSeen(url string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seen[url]
}
