package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/deidaraiorek/deisearch/pkg/textprocessor"
	"github.com/deidaraiorek/deisearch/query-engine/internal/storage"
)

type SearchCache struct {
	mu      sync.RWMutex
	results map[string][]CachedResult
}

type CachedResult struct {
	DocID       int64
	Score       float64 `json:"-"` // Used internally for sorting, not sent to client
	URL         string
	Title       string
	Description string
	Content     string
}

var (
	cache        = &SearchCache{results: make(map[string][]CachedResult)}
	indexReader  *storage.IndexReader
	spiderReader *storage.SpiderReader
)

func InitReaders(indexDBPath, spiderDBPath, pagerankDBPath string) error {
	var err error
	indexReader, err = storage.NewIndexReader(indexDBPath)
	if err != nil {
		return err
	}

	spiderReader, err = storage.NewSpiderReader(spiderDBPath)
	if err != nil {
		return err
	}

	return nil
}

func generateCacheKey(terms []string) string {
	hash := md5.Sum([]byte(joinTerms(terms)))
	return hex.EncodeToString(hash[:])
}

func joinTerms(terms []string) string {
	result := ""
	for i, term := range terms {
		if i > 0 {
			result += "|"
		}
		result += term
	}
	return result
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing required parameter 'q'", http.StatusBadRequest)
		return
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		parsedPage, err := strconv.Atoi(pageStr)
		if err != nil || parsedPage < 1 {
			http.Error(w, "Invalid page number", http.StatusBadRequest)
			return
		}
		page = parsedPage
	}

	processor := textprocessor.NewTextProcessor()
	terms := processor.Process(query)

	if len(terms) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": []interface{}{},
			"total":   0,
			"page":    page,
		})
		return
	}

	limit := 10
	offset := (page - 1) * limit

	cacheKey := generateCacheKey(terms)

	cache.mu.RLock()
	cachedResults, found := cache.results[cacheKey]
	cache.mu.RUnlock()

	if !found {
		termIDs, err := indexReader.GetTermIDs(terms)
		if err != nil {
			http.Error(w, "Failed to get term IDs", http.StatusInternalServerError)
			return
		}

		searchResults, err := indexReader.SearchDocuments(termIDs)
		if err != nil {
			http.Error(w, "Failed to search documents", http.StatusInternalServerError)
			return
		}

		docIDs := make([]int64, len(searchResults))
		for i, result := range searchResults {
			docIDs[i] = result.DocID
		}

		pages, err := spiderReader.GetPagesByIDs(docIDs)
		if err != nil {
			http.Error(w, "Failed to get page info", http.StatusInternalServerError)
			return
		}

		cachedResults = make([]CachedResult, 0, len(searchResults))
		for _, result := range searchResults {
			if pageInfo, ok := pages[result.DocID]; ok {
				// Use TF-IDF score directly
				finalScore := result.Score

				content := pageInfo.Content
				if len(content) > 300 {
					content = content[:300]
				}
				cachedResults = append(cachedResults, CachedResult{
					DocID:       result.DocID,
					Score:       finalScore,
					URL:         pageInfo.URL,
					Title:       pageInfo.Title,
					Description: pageInfo.Description,
					Content:     content,
				})
			}
		}

		sort.Slice(cachedResults, func(i, j int) bool {
			return cachedResults[i].Score > cachedResults[j].Score
		})

		cache.mu.Lock()
		cache.results[cacheKey] = cachedResults
		cache.mu.Unlock()
	}

	total := len(cachedResults)
	start := offset
	end := offset + limit

	if start >= total {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": []interface{}{},
			"total":   total,
			"page":    page,
		})
		return
	}

	if end > total {
		end = total
	}

	pageResults := cachedResults[start:end]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"results": pageResults,
		"total":   total,
		"page":    page,
	})
}
