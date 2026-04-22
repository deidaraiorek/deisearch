package indexer

import (
	"fmt"
	"log"
	"strings"

	"github.com/deidaraiorek/deisearch/semantic-indexer/internal/embedder"
	"github.com/deidaraiorek/deisearch/semantic-indexer/internal/spider"
	"github.com/deidaraiorek/deisearch/semantic-indexer/internal/storage"
)

type Indexer struct {
	spiderDB     *spider.SpiderDB
	embeddingsDB *storage.EmbeddingsDB
	model        *embedder.Model
	batchSize    int
}

func NewIndexer(spiderDBPath, embeddingsDBPath string, batchSize int) (*Indexer, error) {
	spiderDB, err := spider.NewSpiderDB(spiderDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open spider database: %w", err)
	}

	embeddingsDB, err := storage.NewEmbeddingsDB(embeddingsDBPath)
	if err != nil {
		spiderDB.Close()
		return nil, fmt.Errorf("failed to open embeddings database: %w", err)
	}

	model, err := embedder.NewModel()
	if err != nil {
		spiderDB.Close()
		embeddingsDB.Close()
		return nil, fmt.Errorf("failed to initialize model: %w", err)
	}

	return &Indexer{
		spiderDB:     spiderDB,
		embeddingsDB: embeddingsDB,
		model:        model,
		batchSize:    batchSize,
	}, nil
}

func (idx *Indexer) Close() {
	if idx.model != nil {
		idx.model.Close()
	}
	if idx.embeddingsDB != nil {
		idx.embeddingsDB.Close()
	}
	if idx.spiderDB != nil {
		idx.spiderDB.Close()
	}
}

func (idx *Indexer) IndexAll() error {
	totalPages, err := idx.spiderDB.GetTotalPageCount()
	if err != nil {
		return fmt.Errorf("failed to get total page count: %w", err)
	}

	lastIndexedID, err := idx.embeddingsDB.GetLastIndexedPageID()
	if err != nil {
		return fmt.Errorf("failed to get last indexed page ID: %w", err)
	}

	log.Printf("Total pages in spider DB: %d", totalPages)
	log.Printf("Resuming from page ID: %d", lastIndexedID)

	processedCount := 0
	currentID := lastIndexedID

	for {
		pages, err := idx.spiderDB.GetPagesAfterID(currentID, idx.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch pages: %w", err)
		}

		if len(pages) == 0 {
			break
		}

		if err := idx.processBatch(pages); err != nil {
			return fmt.Errorf("failed to process batch: %w", err)
		}

		processedCount += len(pages)
		currentID = pages[len(pages)-1].ID

		log.Printf("Processed %d pages (%.2f%%) - Last ID: %d",
			processedCount,
			float64(processedCount+lastIndexedID)/float64(totalPages)*100,
			currentID)
	}

	if err := idx.embeddingsDB.UpdateMetadata("indexing_complete", "true"); err != nil {
		log.Printf("Warning: failed to update indexing_complete metadata: %v", err)
	}

	log.Printf("Indexing complete! Total pages processed: %d", processedCount)
	return nil
}

func (idx *Indexer) processBatch(pages []*spider.Page) error {
	texts := make([]string, len(pages))
	for i, page := range pages {
		texts[i] = prepareText(page.Title, page.Description, page.Content)
	}

	embeddings, err := idx.model.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	tx, err := idx.embeddingsDB.BeginTransaction()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for i, emb := range embeddings {
		serialized := embedder.SerializeEmbedding(emb)
		if err := idx.embeddingsDB.SaveEmbeddingWithTx(tx, pages[i].ID, pages[i].URL, serialized); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to save embedding for page %d: %w", pages[i].ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	lastID := pages[len(pages)-1].ID
	if err := idx.embeddingsDB.UpdateMetadata("last_indexed_page_id", fmt.Sprintf("%d", lastID)); err != nil {
		log.Printf("Warning: failed to update last_indexed_page_id: %v", err)
	}

	return nil
}

func prepareText(title, description, content string) string {
	var parts []string

	if title != "" {
		parts = append(parts, title)
	}

	if description != "" {
		parts = append(parts, description)
	}

	if content != "" {
		maxContent := 2000
		if len(content) > maxContent {
			content = content[:maxContent]
		}
		parts = append(parts, content)
	}

	return strings.Join(parts, " ")
}
