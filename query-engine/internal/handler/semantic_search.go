package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/deidaraiorek/deisearch/query-engine/internal/embedder"
	"github.com/deidaraiorek/deisearch/query-engine/internal/hnsw"
	"github.com/deidaraiorek/deisearch/query-engine/internal/storage"
)

var (
	embeddingModel   *embedder.Model
	embeddingsReader *storage.EmbeddingsReader
	hnswIndex        *hnsw.Index
	indexDocIDs      []int
)

type SemanticResult struct {
	DocID       int64
	Score       float64
	URL         string
	Title       string
	Description string
	Content     string
}

func InitSemanticSearch(embeddingsDBPath string) error {
	var err error

	embeddingModel, err = embedder.NewModel()
	if err != nil {
		return err
	}

	embeddingsReader, err = storage.NewEmbeddingsReader(embeddingsDBPath)
	if err != nil {
		return err
	}

	log.Printf("Loading embeddings and building HNSW index...")
	embeddings, docIDs, err := embeddingsReader.GetAllEmbeddings()
	if err != nil {
		return err
	}

	hnswIndex = hnsw.NewIndex(384, len(embeddings))
	hnswIndex.SetEf(100)

	indexDocIDs = make([]int, len(docIDs))
	for i, docID := range docIDs {
		indexDocIDs[i] = docID
		hnswIndex.AddPoint(embeddings[i], i)

		if (i+1)%10000 == 0 {
			log.Printf("Built HNSW index: %d/%d (%.1f%%)", i+1, len(docIDs), float64(i+1)/float64(len(docIDs))*100)
		}
	}

	log.Printf("Semantic search initialized successfully - HNSW index with %d embeddings", len(indexDocIDs))
	return nil
}

func HandleSemanticSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondSemanticJSON(w, query, 1, []SemanticResult{}, 0)
		return
	}

	pageParam := r.URL.Query().Get("page")
	page := 1
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	queryEmb, err := embeddingModel.Embed(query)
	if err != nil {
		http.Error(w, "Failed to generate query embedding", http.StatusInternalServerError)
		log.Printf("Error generating query embedding: %v", err)
		return
	}

	neighborIndices := hnswIndex.Search(queryEmb, 100)

	results := make([]SemanticResult, len(neighborIndices))
	for i, idx := range neighborIndices {
		results[i] = SemanticResult{
			DocID: int64(indexDocIDs[idx]),
			Score: 1.0,
		}
	}

	docIDsInt64 := make([]int64, len(results))
	for i, result := range results {
		docIDsInt64[i] = result.DocID
	}

	pages, err := spiderReader.GetPagesByIDs(docIDsInt64)
	if err != nil {
		http.Error(w, "Failed to fetch page metadata", http.StatusInternalServerError)
		log.Printf("Error fetching page metadata: %v", err)
		return
	}

	enrichedResults := make([]SemanticResult, 0, len(results))
	for _, result := range results {
		page, ok := pages[result.DocID]
		if !ok {
			log.Printf("Warning: page %d not found", result.DocID)
			continue
		}

		content := page.Content
		if len(content) > 300 {
			content = content[:300]
		}

		enrichedResults = append(enrichedResults, SemanticResult{
			DocID:       result.DocID,
			Score:       result.Score,
			URL:         page.URL,
			Title:       page.Title,
			Description: page.Description,
			Content:     content,
		})
	}

	paginatedResults, total := paginateSemanticResults(enrichedResults, page, 10)
	respondSemanticJSON(w, query, page, paginatedResults, total)
}

func paginateSemanticResults(results []SemanticResult, page, pageSize int) ([]SemanticResult, int) {
	total := len(results)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return []SemanticResult{}, total
	}

	if end > total {
		end = total
	}

	return results[start:end], total
}

func respondSemanticJSON(w http.ResponseWriter, query string, page int, results []SemanticResult, total int) {
	response := struct {
		Query   string           `json:"query"`
		Results []SemanticResult `json:"results"`
		Total   int              `json:"total"`
		Page    int              `json:"page"`
	}{
		Query:   query,
		Results: results,
		Total:   total,
		Page:    page,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
